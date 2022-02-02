#!/usr/bin/env xonsh
#To run, use xonsh run.xsh  --num <number of nodes> --freq <frequency>
import argparse 
import signal
import time
import os

def parse_opt():
    parser = argparse.ArgumentParser()
    parser.add_argument("--num", type=int, default=3, help="number of nodes")
    parser.add_argument("--freq", type=int, default=10, help="frequency of generator")
    opt = parser.parse_args()
    return opt

def handler(signum, frame):
    res = input("Ctrl-c was pressed. Do you really want to exit? y/n ")
    if res == 'y':
        # kill all jobs
        jobs -x kill
        exit(1)
 
signal.signal(signal.SIGINT, handler)

def main(opt):
    os.chdir("node")
    for i in range(0, opt.num):
        python3 generator.py @(opt.freq) | go run node @(f"node{i}") 127.0.0.1 1234 > /dev/null & 

    os.chdir("../logger")
    go run logger 1234 &

    while(True):
        pass

if __name__ == "__main__":
    opt = parse_opt()
    main(opt)