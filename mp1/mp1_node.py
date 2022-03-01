#!/bin/bash
from platform import node
import socket
import _thread
import threading
import signal
import time

import sys
from collections import defaultdict
#from options import parse_options
import logging as log
from threading import Thread
from account import AccountCtl
from isis import Isis
from message import Message
addr2node = dict()
nodes_event_time = dict()

class node:
    def __init__(self):
        self.node_id = None
        self.proSeq = None
        self.agrSeq = None
        self.node_n = None
        self.all_node_connected = False
        self._set_args()
        self._parse_configuration()
        self._create_socket()
        self.mutex = threading.Lock()
        self.isis_mutex = threading.Lock()
        self.allproposed_mutex = threading.Lock()
        self.recivedDict_mutex = threading.Lock()
        self.agreedDict_mutex = threading.Lock()
        self.acountCtl_mutex = threading.Lock()
        self.connected_node =  set()
        self.acountCtl = AccountCtl()
        self.isis = Isis(self.node_id)
        self.allproposed = defaultdict(list)
        self.recivedDict = defaultdict(int)
        self.agreedDict = defaultdict(int)
        self.broadcast_message = []
        self.unicast_message = []
        # self.holdback = []
        # self.payload = []
        # self.splits = 1

    # get the arguments: node name , logger ip, and logger port
    def _set_args(self):
        try:
            self.identifier = sys.argv[1]
            self.configuration_name = sys.argv[2]
        except:
            print("./mp1_node <identifier> <configuration file>")
            exit(1)

    def _parse_configuration(self):
        try:
            with open(self.configuration_name,'r') as f:
                lines = f.readlines()
                self.node_n = int(lines[0])
                self.nodes_info = [line.strip().split(' ') for line in lines[1:]]
                for i, node_info in enumerate(self.nodes_info):
                    if self.identifier == node_info[0]:
                        self.node_id = i

        except:
            print("can not read the file")
            exit(1)
        

    def _create_socket(self):
        for node_info in self.nodes_info:
            if node_info[0] == self.identifier:
                HOST = ""
                PORT = int(node_info[2])

        self.listen_s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.listen_s.bind((HOST, PORT))
        self.listen_s.listen(1)
        bitmask = [0]*len(self.nodes_info)

        self.send_s = defaultdict()
        while sum(bitmask) != len(bitmask):
            for i in range(len(bitmask)):
                if bitmask[i] == 1:
                    continue
                
                # try to connect to 
                try:
                    node_info = self.nodes_info[i]
                    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                    IP_addr = socket.gethostbyname(node_info[1])
                    s.connect((IP_addr, int(node_info[2])))
                    self.send_s[i]= s
                    bitmask[i] = 1
                    print("connect to ", node_info[0])
                except:
                    continue
            time.sleep(2)
            
        self.b_broadcast(f"{self.identifier} TCP connected")

    
    def b_broadcast(self, message):
        for node_id in self.send_s.keys():
            self.send_s[node_id].sendall(bytes(f'{message}', "UTF-8"))
    
    def unicast(self, message, target_id):
        self.send_s[target_id].sendall(bytes(f'{message}', "UTF-8"))
    
    def listen(self):
        conn, addr = self.listen_s.accept()
        with conn:
            # loop until all the nodes have connected to other nodes
            
            data = conn.recv(1024)
            data = data.decode('utf-8').split(' ')[0]
            self.mutex.acquire()
            self.connected_node.add(data)
            print(self.connected_node)
            self.mutex.release()

            while True:
                self.mutex.acquire()
                if self.all_node_connected or len(self.connected_node) == self.node_n:
                    self.all_node_connected = True
                    self.mutex.release()
                    self.all_node_connected = True
                    print("all node conected")
                    break
                else:
                    self.mutex.release()
                time.sleep(1)
            
            time.sleep(5)
            while True:
                # listen messages from other nodes
                messages = conn.recv(4096).decode('utf-8')
                if messages == '':
                    continue
                messages = messages.strip().split('\n')
                
                for message in messages:
                    msg = Message(message)

                    if msg.isis_type == 'MESSAGE' and msg.id not in self.recivedDict.keys():
                        self.recivedDict_mutex.acquire()
                        if self.recivedDict[msg.id] == 0:
                            self.recivedDict_mutex.release()
                            # R-multicast implementation
                            sender_id = msg.node_id
                            msg.node_id = self.node_id
                            self.b_broadcast(msg.construct_string())

                            # unicast the priority
                            self.isis_mutex.acquire()
                            proposed = self.isis.proposeSeq(msg)
                            self.isis_mutex.release()

                            self.recivedDict_mutex.acquire()
                            self.recivedDict[msg.id] = 1
                            self.recivedDict_mutex.release()
                            
                            msg.priority = proposed
                            msg.isis_type = 'PROPOSE'
                            self.unicast(msg.construct_string(),sender_id)
                            
                            # record the message
                        else:
                            self.recivedDict_mutex.release()
                    
                        
                        
                    elif msg.isis_type == 'PROPOSE':
                        self.allproposed_mutex.acquire()
                        self.allproposed[msg.id].append(msg)
                        if len(self.allproposed[msg.id]) == self.node_n:
                            
                            # get the agreed priority using isis algorithm
                            self.isis_mutex.acquire()
                            decided_seq,message_id = self.isis.decideSeq(self.allproposed[msg.id])
                            self.isis_mutex.release()
                            self.allproposed[msg.id] = []
                            self.allproposed_mutex.release()

                            # send the message with agreed priority
                            msg.priority = decided_seq
                            msg.isis_type = 'AGREE'
                            self.b_broadcast(msg.construct_string())
                        else:
                            self.allproposed_mutex.release()
                        
                    elif msg.isis_type == 'AGREE':
                        self.agreedDict_mutex.acquire()
                        if self.agreedDict[msg.id] == 0: 
                            self.agreedDict[msg.id] = 1
                            self.agreedDict_mutex.release()

                            # get the deliverable messages
                            self.isis_mutex.acquire()
                            deliverMsgs = self.isis.deliverMsg(msg)
                            self.isis_mutex.release()

                            # deliver the messages
                            for deliver_msg in deliverMsgs:
                                self.acountCtl_mutex.acquire()
                                self.acountCtl.updateBalance(deliver_msg)
                                self.acountCtl_mutex.release()
                        else:
                            self.agreedDict_mutex.release()



    
    def send(self):
        while not self.all_node_connected:
            time.sleep(1)
        time.sleep(5)
        for line in sys.stdin: 
            msg = Message(line)
            self.isis_mutex.acquire()    
            proposed = self.isis.proposeSeq(msg)
            self.isis_mutex.release()

            msg.priority = proposed
            msg.node_id = self.node_id
            self.allproposed_mutex.acquire()
            self.allproposed[msg.id].append(msg)
            self.allproposed_mutex.release()

            self.b_broadcast(msg.construct_string())
            


if __name__ == "__main__":
    # node_n: int, nodes_info [node, 3],  [id, ip_name, port]
    node = node()
    for i in range(node.node_n):
        handleRequest = threading.Thread(target=node.listen,args=())
        handleRequest.start()
    
    sending_threads = threading.Thread(target=node.send,args=())
    sending_threads.start()
