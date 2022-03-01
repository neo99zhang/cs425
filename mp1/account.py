from collections import defaultdict
from message import Message
class AccountCtl:
    def __init__(self):
        self.balance = defaultdict(int)

    def updateBalance(self, msg):
        if msg.type ==  'DEPOSIT':
            self.balance[msg.source] += msg.amount 
            self.printAccount()
        elif msg.type == 'TRANSFER':
            if self.balance[msg.source] >= msg.amount:
                self.balance[msg.source] -= msg.amount
                self.balance[msg.target] += msg.amount
                self.printAccount()
        
    
    def printAccount(self):
        accounts = self.balance.keys()
        accounts.sort()
        out = "BALANCES"
        for account in accounts:
            out += f" {account}:{self.balance[account]}"
        out += '\n'
        print(out)

