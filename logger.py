import socket
import _thread
import threading
import signal
import time
import sys

class Node:

    def __init__(self):
        self.id = -1 
        self.payload = []
        self.splits = 1

    def addToNode(self,inputID):
        self.id = inputID
    
    def addToPayload(self,time,event):
        if len(self.payload) < 1:
            pass
        #TODO actually I am confused how to deal with the payload input...
        else:
        #TODO
            self.payload[-1].append(time.time())
            self.payload[-1].append(event)


class Cluster:

    def __init__(self):
        self.ndict = {}

    def addToIP(self,addr,nid):
        self.ndict[addr] = nid


if __name__ == '_main_':
    '''
    The main loop should be running and serving as logger.
    '''
