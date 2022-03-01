import hashlib
import time

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
        self.node_id = node_id
        
        self.process_message()
        '''
        Type:DEPOSITE, TRANSFER,
        '''
    
    def process_message(self):
        message = self.message.strip().split(' ')
        # read from terminal
        if message[0] == 'DEPOSIT':
            self.type = 'DEPOSIT'
            self.source = message[1]
            self.amount = int(message[2])
            self.isis_type = 'MESSAGE'
            self.id = time.time()
            self.node_id = int(message[5])

        # read from terminal
        elif message[0] == 'TRANSFER':
            self.type = 'TRANSFER'
            self.source = message[1]
            self.target = message[3]
            self.amount = int(message[4])
            self.isis_type = 'MESSAGE'
            self.id = time.time()
            self.node_id = int(message[6])

        # read from other nodes
        else:
            if message[2] == 'DEPOSIT':      
                self.isis_type = message[0]
                self.id = message[1]
                self.type = message[2]
                self.source = message[3]
                self.amount = message[4]
                self.priority = message[5]
                self.node_id = message[6]
            elif message[2] == 'TRANSFER':
                self.isis_type = message[0]
                self.id = message[1]
                self.type = message[2]
                self.source = message[3]
                self.target = message[4]
                self.amount = message[5]
                self.priority = message[6]
                self.node_id = message[7]

                
            

    def construct_string(self):
        if self.isis_type == 'MESSAGE':
            if self.type == 'DEPOSIT':
                return f'{self.isis_type} {self.id} DEPOSIT {self.source} {self.amount} {self.node_id}'
            elif self.type == 'TRANSFER':
                return f'{self.isis_type} {self.id} TRANSFER {self.source} {self.target} {self.amount} {self.node_id}'
            
        else:
            if self.type == 'DEPOSIT':
                return f'{self.isis_type} {self.id} DEPOSIT {self.source} {self.amount} {self.priority} {self.node_id}'
            elif self.type == 'TRANSFER':
                return f'{self.isis_type} {self.id} TRANSFER {self.source} {self.target} {self.amount} {self.priority} {self.node_id}'
