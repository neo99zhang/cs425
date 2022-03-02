import heapq
import bisect 
class KeyWrapper:
    def __init__(self, iterable, key):
        self.it = iterable
        self.key = key

    def __getitem__(self, i):
        return self.key(self.it[i])

    def __len__(self):
        return len(self.it)

class Isis:
    def __init__(self, node_id):
        self.queue = []
        self.proSeq = float(node_id)*0.1
        self.agrSeq = float(node_id)*0.1

    def proposeSeq(self,Msg):
        '''
        input: a process
        output: proposed seq num
        '''
        #p.agrSeq += self.counter
        self.proSeq = max(self.proSeq,self.agrSeq) + 1.0
        Msg.deliverable = False
        Msg.priority = self.proSeq
        bslindex = bisect.bisect_left(KeyWrapper(self.queue,key = lambda x:x[0]),self.proSeq)
        self.queue.insert(bslindex,(self.proSeq, Msg))

        return self.proSeq

    def decideSeq(self, ListMsg):
        '''
        input: a list of Msg, each with message id and proposed seq num
        output: agreed seq num, the message id selected
        '''
        max_msg = max(ListMsg, key=lambda x:x.priority)
        max_priority = max_msg.priority
        self.agrSeq = max(self.agrSeq,max_priority)
        return max_priority

    def deliverMsg(self,Msg):
        deliverMsgs = []
        # update the priority of the coming message
        for i,pair in enumerate(self.queue):
            m = pair[1]
            if m.id == Msg.id:
                Msg.deliverable = True
                self.queue[i] = self.queue[-1]
                self.queue.pop()
                if i < len(self.queue):
                    heapq._siftup(self.queue, i)
                    heapq._siftdown(self.queue, 0, i)
                heapq.heappush(self.queue,(Msg.priority,Msg))
                break

        # deliver all the avaliable messages
        while not (self.queue == []):
            m = self.queue[0][1]
            if m.deliverable:
                self.queue.pop(0)
                deliverMsgs.append(m)
            else:
                break
        for pair in self.queue:
            print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0])
        return deliverMsgs

