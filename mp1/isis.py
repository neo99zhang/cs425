import heapq

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
        heapq.heappush(self.queue,(self.proSeq,Msg))
        return self.proSeq

    def decideSeq(self, ListMsg):
        '''
        input: a list of Msg, each with message id and proposed seq num
        output: agreed seq num, the message id selected
        '''
        id = ListMsg[0].id
        max_msg = max(ListMsg, key=lambda x:x.priority)
        max_priority = max_msg.priority
        return max_priority, id

    def deliverMsg(self,Msg):
        deliverMsgs = []
        # update the priority of the coming message
        for pair in self.queue:
            m = pair[1]
            if m.id == Msg.id:
                Msg.deliverable = True
                self.queue.remove(pair)
                heapq.heappush(self.queue,(Msg.priority,Msg))
                break

        # deliver all the avaliable messages
        while not (self.queue == []):
            pair = self.queue.pop(0)
            m = pair[1]
            if m.deliverable is False:
                heapq.heappush(self.queue,pair)
                break
            else:
                deliverMsgs.append(m)
        return deliverMsgs

