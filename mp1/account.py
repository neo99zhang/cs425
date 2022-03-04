from collections import defaultdict
from message import Message
import time
class AccountCtl:
    def __init__(self):
        self.balance = defaultdict(int)

    def updateBalance(self, msg):
        #get transaction time:
        with open('transaction.txt', 'wt') as f:
            trans_time = time.time() - msg.id
            print('Transaction: ',trans_time,"Transaction time: ",msg.id,file=f)
        if msg.type ==  'DEPOSIT':
            self.balance[msg.source] += msg.amount 
            self.printAccount()
        elif msg.type == 'TRANSFER':
            if self.balance[msg.source] >= msg.amount:
                self.balance[msg.source] -= msg.amount
                self.balance[msg.target] += msg.amount
                self.printAccount()
        
    
    def printAccount(self):
        accounts = list(self.balance.keys())
        accounts.sort()
        out = "BALANCES"
        for account in accounts:
            if self.balance[account] == 0:
                continue
            out += f" {account}:{self.balance[account]}"

        print(out)

