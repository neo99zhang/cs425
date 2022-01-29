import socket
import _thread
import threading
import signal
import time
import sys
from options import parse_options
import logging as log
from threading import Thread


class Cluster:
    def __init__(self,args):
        self.ndict = {}

    def addToIP(self,addr,nid):
        self.ndict[addr] = nid
    
    


if __name__ == '__main__':
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(('localhost', 50000))
    s.listen(1)
    conn, addr = s.accept()
    while 1:
        data = conn.recv(1024)
        if not data:
            break
        conn.sendall(data)
    conn.close()
    '''
    The main loop should be running and serving as logger.
    '''
