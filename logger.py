import socket
import _thread
import threading
import signal
import time
import sys
from options import parse_options
import logging as log
from threading import Thread


class Logger:
    def __init__(self):
        self.ndict = {}
        self._set_args()
        self._TCP_connect()

    def addToIP(self,addr,nid):
        self.ndict[addr] = nid
    
    def _set_args(self):
        try:
            self.port = int(sys.argv[1])
        except:
            print("Please use ./logger <port>")
            exit(1)
    
    def _TCP_connect(self):
        self.s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.s.bind(('172.30.124.143', self.port))
        self.s.listen(1)
        self.conn, addr = self.s.accept()
    
    def read(self):
        while 1:
            data = str(self.conn.recv(1024)).split(' ')
            time_stamp = data[0][2:] # e.g 1643485243.730725
            content = data[1][:-2] # e.g fca892488ee6f38ff20fde9720056dc9c454c680b5aef171036fe0468f81fc08
            node_name = data[2][:-1] # e.g node1
            print(f'{time_stamp} {node_name} {content}')

if __name__ == '__main__':
    logger = Logger()
    logger.read()
    
    '''
    The main loop should be running and serving as logger.
    '''
