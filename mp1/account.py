from collections import defaultdict
class Account:
    def __init__(self):
        self.balance = defaultdict(int)

    def updateBalance(self, message):
        message = message.strip().split(' ')
        if message[0] == 'DEPOSITE':
            account = message[1]
            amount = message[2]
            self.balance[account] += amount 
        elif message[0] == 'TRANSFER':
            sender = message[1]
            receiver = message[3]
            amount = message[4]
            if self.balance[sender] >= amount:
                self.balance[sender] -= amount
                self.balance[receiver] += amount
        
    
    def printAccount(self):
        accounts = self.balance.keys()
        accounts.sort()
        out = "BALANCES"
        for account in accounts:
            out += f" {self.balance[account]}"
        out += '\n'
        print(out)

