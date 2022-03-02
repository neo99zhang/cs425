import hashlib
import time
import sys

class Message:
    def __init__(self,message,node_id=None):
        self.message = message
        self.type = None
        self.source = None
        self.target = None
        self.amount = None
        self.deliverable = False
        self.isis_type = None
        self.priority = None
        self.id = None
        self.node_id = int(node_id) if node_id != None else None
        
        self.process_message()
        '''
        Type:DEPOSITE, TRANSFER,
        '''
    
    def process_message(self):
        message = self.message.strip().split(' ')
        # read from terminal
        try:
            if message[0] == 'DEPOSIT':
                self.type = 'DEPOSIT'
                self.source = message[1]
                self.amount = int(message[2])
                self.isis_type = 'MESSAGE'
                self.id = time.time()

            # read from terminal
            elif message[0] == 'TRANSFER':
                self.type = 'TRANSFER'
                self.source = message[1]
                self.target = message[3]
                self.amount = int(message[4])
                self.isis_type = 'MESSAGE'
                self.id = time.time()

            # read from other nodes
            else:
                self.isis_type = message[0]
                self.id = message[1]
                self.type = message[2]
                self.source = message[3]
                if message[0] == 'MESSAGE':
                    self.source = message[3]
                    if message[2] == 'DEPOSIT':
                        self.amount = int(message[4])
                        self.node_id = int(message[5])
                    else:
                        self.target = message[4]
                        self.amount = int(message[5])
                        self.node_id = int(message[6])
                elif message[2] == 'DEPOSIT':      
                    self.amount = int(message[4])
                    self.priority = float(message[5])
                    self.node_id = int(message[6])
                elif message[2] == 'TRANSFER':
                    self.target = message[4]
                    self.amount = int(message[5])
                    self.priority = float(message[6])
                    self.node_id = int(message[7])
        except:
            print("message error: ", message)
            sys.exit(0)

                
            

    def construct_string(self):
        if self.isis_type == 'MESSAGE':
            if self.type == 'DEPOSIT':
                s =  f'{self.isis_type} {self.id} DEPOSIT {self.source} {self.amount} {self.node_id}'
            elif self.type == 'TRANSFER':
                s =  f'{self.isis_type} {self.id} TRANSFER {self.source} {self.target} {self.amount} {self.node_id}'
            
        else:
            if self.type == 'DEPOSIT':
                s =  f'{self.isis_type} {self.id} DEPOSIT {self.source} {self.amount} {self.priority} {self.node_id}'
            elif self.type == 'TRANSFER':
                s = f'{self.isis_type} {self.id} TRANSFER {self.source} {self.target} {self.amount} {self.priority} {self.node_id}'
        s += (256 - len(s))*' '
        return s
