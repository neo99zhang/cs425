class Isis:
    def __init__(self):
        self.counter = 0.0

    def proposeSeq(self,p):
        '''
        input: a process
        output: proposed seq num
        '''
        if p.proSeq is None or p.agrSeq is None:
            print("Now proposing Seq Num"+str("0."+p.pid))
            seq = self.counter + float(p.pid)*0.1
            p.proSeq = seq
            p.agrSeq = seq
            self.counter += 1.0
            return seq
        self.counter += 1.0
        p.proSeq += self.counter
        p.agrSeq += self.counter
        seq = max(p.proSeq,p.agrSeq) + self.counter
        return seq

    def decideSeq(ListProcess):
        tempMax = 0.0
        for proc in ListProcess:
            if proc.proSeq > tempMax:
                tempMax = proc.ProSeq
        return (tempMax + 1.0)

    def storeMsg(p,msg):
        p.holdback.append(float(str(msg.proSeq)))
