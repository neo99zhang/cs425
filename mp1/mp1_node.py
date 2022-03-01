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
        self._set_args()
        self._parse_configuration()
        self._create_socket()
        self.mutex = threading.Lock()
        self.connected_node =  set()
        self.acountCtl = AccountCtl()
        self.isis = Isis()
        self.all_node_connected = False
        self.node_id = None
        self.proSeq = None
        self.agrSeq = None
        self.node_n = None
        self.allproposed = defaultdict(list())
        self.recivedDict = defaultdict(int)
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
        print(HOST, PORT)
        bitmask = [0]*len(self.nodes_info)
        bitmask[self.node_id] =1

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
                    self.send_s[node_info[0]]= s
                    bitmask[i] = 1
                    print("connect to ", node_info[0])
                except:
                    continue
            time.sleep(2)
            
        self.b_broadcast("TCP connected")

    
    def b_broadcast(self, message):
        for node_id in self.send_s.keys():
            self.send_s[node_id].sendall(bytes(f'{self.identifier} {message}', "UTF-8"))
    
    def listen(self):
        conn, addr = self.listen_s.accept()
        with conn:
            # loop until all the nodes have connected to other nodes
            while True:
                self.mutex.acquire()
                if len(self.connected_node) == self.node_n:
                    self.all_node_connected = True
                    self.mutex.release()
                    print("all node conected")
                    break
                self.mutex.release()
                data = conn.recv(1024)
                data = data.decode('utf-8').split(' ')[0]
                self.mutex.acquire()
                self.connected_node.add(data)
                print(self.connected_node)
                self.mutex.release()
                    
            # listen messages from other nodes
            message = conn.recv(1024).decode('utf-8')
            msg = Message(message)

            self.mutex.acquire()
            if msg.isis_type == 'MESSAGE' and msg.id not in self.recivedDict.keys():
                #propose
                ##TODO
                self.broadcast_message.append(msg)
                proposed = self.isis.proposeSeq(msg)
                msg.priority = proposed
                msg.isis_type = 'PROPOSE'
                # record the message
                self.recivedDict[msg.id] = 1
                
            elif msg.isis_type == 'PROPOSE':
                #agree
                self.allproposed[msg.id].append[msg]
                if len(self.allproposed[msg.id]) == self.node_n:
                    decided_seq,message_id = self.isis.decideSeq(self.allproposed[msg.id])
                    self.allproposed[msg.id] = []
                    msg.priority = decided_seq
                    self.broadcast_message.append(msg)
                
            elif msg.isis_type == 'AGREE':
                deliverMsgs = self.isis.deliverMsgs(msg)
                for deliver_msg in deliverMsgs:
                    self.acountCtl.updateBalance(deliver_msg)
            else:
                print("Error, none of seq type got")
            self.mutex.release()


    
    def send(self):
        while not self.all_node_connected:
            time.sleep(1)
        for line in sys.stdin: 
            print('sending thread,',line)
            msg = Message(line)
            self.mutex.acquire()    
            proposed = self.isis.proposeSeq(msg)
            msg.priority = proposed
            self.allproposed[msg.id].append(msg)
            self.b_broadcast(line.strip()+f" {self.node_id}")
            
            # broadcast message
            while self.broadcast_message != []:
                msg = self.broadcast_message.pop(0)
                msg.node_id = self.node_id
                self.b_broadcast(msg.construct_string())
            
            # unicast message
            while self.unicast_message != []:
                msg = self.unicast_message.pop(0)
                target_id = msg.node_id
                msg.node_id = self.node_id
                self.send_s[target_id].sendall(bytes(msg.construct_string(), "UTF-8"))

            self.mutex.release()
            # print(f'Sending : {send_data} to Cluster')


if __name__ == "__main__":
    # node_n: int, nodes_info [node, 3],  [id, ip_name, port]
    node = node()
    for i in range(node.node_n):
        handleRequest = threading.Thread(target=node.listen,args=())
        handleRequest.start()
    
    sending_threads = threading.Thread(target=node.send,args=())
    sending_threads.start()
