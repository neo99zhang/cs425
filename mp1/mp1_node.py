#!/bin/bash
from platform import node
import socket
import _thread
import threading
import signal
import time
import os
import sys
from collections import defaultdict
#from options import parse_options
import logging as log
log.basicConfig(filename="config.log",filemode="a",format="%(asctime)s-%(name)s-%(levelname)s-%(message)s",level=log.INFO)
log.info('info')
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
        self.senderlock = None
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
        self.identifier2id = defaultdict(int)
        self.deadnode = []

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
                    
            self.senderlock = [threading.Lock() for _ in range(self.node_n)]

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
        self.listen_s.listen(32)
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
                    #self.recvlock[i].append(threading.lock())
                    #self.node2idx[s] = i
                    bitmask[i] = 1
                    print("connect to ", node_info[0])
                except:
                    continue
            time.sleep(2)
            
        self.b_broadcast(f"{self.identifier} TCP connected")

    
    def b_broadcast(self, message):
        # print(" b-cast: ",self.node_id, ' ', message)
        for node_id in self.send_s.keys():
            try:
                self.unicast(message,node_id)
            except:
                print("error message: ", message)
                print("length of message: ", len(message))
                exit(1)
    
    def unicast(self, message, target_id):
        with self.senderlock[target_id]:
            try:
                self.send_s[target_id].sendall(bytes(f'{message}', "UTF-8"))
            except:
                pass
    
    
    def listen(self):
        conn, addr = self.listen_s.accept()

        for i, node_info in enumerate(self.nodes_info):
            IP_address = socket.gethostbyname(node_info[1])
            if IP_address == addr[0]:
                connected_node_id = i

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
            
            #time.sleep(5)
            while True:
                # listen messages from other nodes
                #messages = conn.recv().decode('utf-8')
                #message = conn.recv(256).decode('utf-8')
                conn.settimeout(4)
                try:
                    message = conn.recv(256).decode('utf-8')
                except conn.timeout:
                    self.deadnode.append(connected_node_id)
                    #if node dead
                    # delete the related entries in the queue
                    print("before isis delete")
                    with self.isis_mutex:
                        
                        self.isis.delete_node(connected_node_id)
                    # decrese the number of nodes
                    with self.senderlock[connected_node_id]:
                        self.send_s.pop(connected_node_id)
                    self.node_n -= 1

                    break

                # if not message:
                #     self.deadnode.append(connected_node_id)
                #     #if node dead
                #     # delete the related entries in the queue
                #     print("before isis delete")
                #     with self.isis_mutex:
                        
                #         self.isis.delete_node(connected_node_id)
                #     # decrese the number of nodes
                #     with self.senderlock[connected_node_id]:
                #         self.send_s.pop(connected_node_id)
                #     self.node_n -= 1

                #     break
                message = message.strip()
                # if message == '':
                #     time.sleep(0.2)
                #     continue

                if connected_node_id in self.deadnode:
                    continue

                msg = Message(message)

                # self.mutex.acquire()
                if msg.isis_type == 'MESSAGE':
                    self.recivedDict_mutex.acquire()
                    if self.recivedDict[msg.id] == 0:
                        self.recivedDict[msg.id] = 1
                        # print("get: ", msg.construct_string().strip())
                        self.recivedDict_mutex.release()
                        # R-multicast implementation
                        sender_id = msg.node_id
                        self.b_broadcast(msg.construct_string())

                        # unicast the priority
                        self.isis_mutex.acquire()
                        proposed = self.isis.proposeSeq(msg)
                        self.isis_mutex.release()
                        
                        msg.priority = proposed
                        msg.isis_type = 'PROPOSE'
                        
                        self.unicast(msg.construct_string(),sender_id)
                        log.info(f"SEND: {msg.construct_string().strip()}")
                        # record the message
                    else:
                        self.recivedDict_mutex.release()
                
                    
                    
                elif msg.isis_type == 'PROPOSE':
                    self.allproposed_mutex.acquire()
                    self.allproposed[msg.id].append(msg)
                    #print("The msg is",msg.id," And got",len(self.allproposed[msg.id]),"propose until now")
                    # print("get: ", msg.construct_string().strip())
                    if len(self.allproposed[msg.id]) == self.node_n:
                        #print("The msg is",msg.id," And got",len(self.allproposed[msg.id]),"propose until now, which is enough")
                        log.info(f"GET: {msg.construct_string().strip()}")
                        # get the agreed priority using isis algorithm
                        self.isis_mutex.acquire()
                        decided_seq = self.isis.decideSeq(self.allproposed[msg.id])
                        self.isis_mutex.release()
                        self.allproposed[msg.id] = []
                        self.allproposed_mutex.release()

                        # send the message with agreed priority
                        msg.priority = decided_seq
                        msg.isis_type = 'AGREE'
                        # print("send: ", msg.construct_string().strip())
                        log.info(f"Send: {msg.construct_string().strip()}")
                        self.b_broadcast(msg.construct_string())
                    else:
                        self.allproposed_mutex.release()
                    
                elif msg.isis_type == 'AGREE':
                    self.agreedDict_mutex.acquire()
                    if self.agreedDict[msg.id] == 0: 
                        # print("get: ", msg.construct_string().strip())
                        log.info(f"GET: {msg.construct_string().strip()}")
                        self.agreedDict[msg.id] = 1
                        self.agreedDict_mutex.release()
                        self.b_broadcast(msg.construct_string())
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

                # self.mutex.release()



    
    def send(self):
        while not self.all_node_connected:
            time.sleep(1)
        time.sleep(5)
        for line in sys.stdin: 
            msg = Message(line)
            msg.node_id = self.node_id
            # print("send: ", msg.construct_string().strip())
            log.info(f"SEND: {msg.construct_string().strip()}")
            self.b_broadcast(msg.construct_string())
            


if __name__ == "__main__":
    # node_n: int, nodes_info [node, 3],  [id, ip_name, port]
    # os.remove(r"transaction.txt")
    my_node = node()
    for i in range(my_node.node_n):
        handleRequest = threading.Thread(target=my_node.listen,args=())
        handleRequest.start()
    print(my_node.senderlock)
    sending_threads = threading.Thread(target=my_node.send,args=())
    sending_threads.start()
