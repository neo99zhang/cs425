import heapq
import bisect 
import logging as log
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
        # print("before the push")
        # for pair in self.queue:
        #    print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0])
        heapq.heappush(self.queue,(Msg.priority,Msg))
        # print("Just push","priority:Msg",Msg.priority,":",Msg.id)
        # for pair in self.queue:
        #    print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0])


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
        # print("going to push the agreed")
        # for pair in self.queue:
        #    print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0])

        # print("Now going to put agreed",Msg.id,"with priority",Msg.priority)
        for i,pair in enumerate(self.queue):
            m = pair[1]
            if m.id == Msg.id:
                # print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with proposed priotiy",pair[0],"now we change it to",Msg.priority)
                Msg.deliverable = True
                self.queue[i] = self.queue[-1]
                self.queue.pop()
                if i < len(self.queue):
                    # heapq._siftup(self.queue, i)
                    # heapq._siftdown(self.queue, 0, i)
                    try:
                        heapq._siftup(self.queue, i)
                        heapq._siftdown(self.queue, 0, i)
                    except:
                        for i,pair in enumerate(self.queue):
                            for j,pair2 in enumerate(self.queue):
                                if pair2[0] == pair[0] and i != j:
                                    print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0],"msg",pair[1].construct_string().strip())
                                    print("The msg is",pair2[1].id,"and it's deliverable status is:",pair2[1].deliverable," with priotiy",pair2[0],"msg",pair2[1].construct_string().strip())  
                try:
                    heapq.heappush(self.queue,(Msg.priority,Msg))
                except:
                    for i,pair in enumerate(self.queue):
                        for j,pair2 in enumerate(self.queue):
                            if pair2[0] == pair[0] and i != j:
                                print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0],"msg",pair[1].construct_string().strip())
                                print("The msg is",pair2[1].id,"and it's deliverable status is:",pair2[1].deliverable," with priotiy",pair2[0],"msg",pair2[1].construct_string().strip())      
                break
        #print("before the deliver")
        


        # deliver all the avaliable messages
        while not (self.queue == []):
            m = self.queue[0][1]
            if m.deliverable:
                try:
                    heapq.heappop(self.queue)
                except:
                    for i,pair in enumerate(self.queue):
                        for j,pair2 in enumerate(self.queue):
                            if pair2[0] == pair[0] and i != j:
                                print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0],"msg",pair[1].construct_string().strip())
                                print("The msg is",pair2[1].id,"and it's deliverable status is:",pair2[1].deliverable," with priotiy",pair2[0],"msg",pair2[1].construct_string().strip())                       
                deliverMsgs.append(m)
            else:
                break

        #for pair in self.queue:
        #   log.info(f"    {pair[1].id} {pair[1].priority} {pair[1].deliverable}")
        #print("after the deliver")
        # for pair in self.queue:
        #     print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with priotiy",pair[0])
        return deliverMsgs
    
    def delete_node(self,node_id):
        print('enter delete_node')
        print('the queue size is ', len(self.queue))
        for pair in self.queue:
            print("The msg is",pair[1].id,"and it's deliverable status is:",pair[1].deliverable," with proposed priotiy",pair[0],"now we change it to")
                
        for i,pair in enumerate(self.queue):
            m = pair[1]
            if (m.node_id == node_id) and (m.deliverable == False):
                self.queue[i] = self.queue[-1]
                self.queue.pop()
                if i < len(self.queue):
                    heapq._siftup(self.queue, i)
                    heapq._siftdown(self.queue, 0, i)
        for pair in self.queue:
            print("   ",pair[1].construct_string().strip())
            
