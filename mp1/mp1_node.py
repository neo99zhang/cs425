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

addr2node = dict()
nodes_event_time = dict()

class node:
    def __init__(self):
        self._set_args()
        self._parse_configuration()
        self._create_socket()
        self.mutex = threading.Lock()
        self.connected_node =  set()
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

        self.send_s = defaultdict()
        while sum(bitmask) != len(bitmask):
            print (bitmask)
            for i in range(len(bitmask)):
                if bitmask[i] == 1:
                    continue
                if self.identifier == self.nodes_info[i][0]:
                    bitmask[i] =1
                    continue
                
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
    
    def run(self):
        conn, addr = self.listen_s.accept()
        with conn:
            # loop untill connected all the nodes
            while True:
                self.mutex.acquire()
                if len(self.connected_node) == self.node_n -1:
                    self.mutex.release()
                    print("all node conected")
                    break
                self.mutex.release()
                data = conn.recv(1024)
                data = data=data.decode('utf-8').split(' ')[0]
                self.mutex.acquire()
                self.connected_node.add(data)
                print(self.connected_node)
                self.mutex.release()
            



if __name__ == "__main__":
    # node_n: int, nodes_info [node, 3],  [id, ip_name, port]
    node = node()
    thread = []
    for i in range(node.node_n):
        handleRequest = threading.Thread(target=node.run,args=())
        handleRequest.start()
        
