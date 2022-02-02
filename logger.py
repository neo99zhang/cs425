import socket
import _thread
import threading
import signal
import time
import sys
#from options import parse_options
import logging as log
from threading import Thread

addr2node = dict()
threads = []

def create_socket():
    HOST = ''
    PORT = 1234
    try:
        PORT = int(sys.argv[1])
    except:
        print("Please use ./logger <port>")
        exit(1)

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind((HOST, PORT))
    s.listen(1)
    return s

def TCP_connect(s):
    conn, addr = s.accept()
    addr = addr[0]
    data = conn.recv(1024).decode('utf-8').split(' ')
    time_stamp = data[0]
    node_name = data[1]
    if addr not in addr2node.keys():
        addr2node[addr] = node_name

    print(f'{time_stamp} - {node_name} connected')
    return conn

def read(conn):
    with conn:
        while 1:
            data = conn.recv(1024).decode('utf-8').split(' ')
            if not len(data) == 3:
                break 
            time_stamp = data[0] # e.g 1643485243.730725
            content = data[1] # e.g fca892488ee6f38ff20fde9720056dc9c454c680b5aef171036fe0468f81fc08
            node_name = data[2] # e.g node1
            print(f'{time_stamp} {node_name} {content}')            
        conn.close()
    print(f'{time.time()} - {node_name} disconnected')      

    
if __name__ == '__main__':
    s = create_socket()

    while True:
        conn = TCP_connect(s)
        handleRequest = threading.Thread(target=read,args=(conn,))
        handleRequest.start()
        threads.append(handleRequest)
    #s.close() never going to use this line
    '''
    The main loop should be running and serving as logger.
    '''
