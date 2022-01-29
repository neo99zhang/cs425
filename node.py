import sys
import socket

class Node:
    def __init__(self):
        self._set_args()
        self._TCP_connect()
        # self.payload = []
        # self.splits = 1

    # get the arguments: node name , logger ip, and logger port
    def _set_args(self):
        try:
            self.name = sys.argv[1]
            self.logger_ip = sys.argv[2]
            self.logger_port = int(sys.argv[3])
        except:
            print("Please use ./node <name> <logger ip> <logger port>")
            exit(1)

    # Use TCP protocol to connect the logger
    def _TCP_connect(self):
        self.s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.s.connect((self.logger_ip, self.logger_port))
        self.s.sendall(bytes(self.name))

    def read_send(self):
        # read the generator and send the data to logger.
        # data = sys.stdout.readline() ?
        # s.sendall(bytes(data, "UTF-8"))
        for line in sys.stdin:
            send_data = f'{line} {self.name}'
            self.s.sendall(bytes(send_data,"UTF-8"))
            print(f'Sending : {send_data} to Cluster')

if __name__ == '__main__':
    node = Node()
    node.read_send()