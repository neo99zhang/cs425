import os 
import glob
from collections import defaultdict
import numpy as np
import matplotlib.pyplot as plt 
import numpy as np 


logs_3 = glob.glob('3Node_2HZ/*')
logs_8 = glob.glob('8Node_5HZ/*')
NODE3_START = 1643862300
NODE8_START = 1643862750
TIME_PERIOD = 100

delay_dict = defaultdict(list)
band_width_dict = defaultdict(int)
def get_delay_bandwidth(logs, start):
    for log in logs:
        file = open(log)
        for data in file.readlines():
            data = data.split(' ')
            time_stamp_new = float(data[0])
            time_stamp = float(data[1])
            length = int(data[3])
            if time_stamp < start:
                continue   
            if time_stamp > start +TIME_PERIOD:
                break
            delay = time_stamp_new - time_stamp
            delay_dict[int(time_stamp)].append(delay)
            band_width_dict[int(time_stamp)] += length*8
    mean_delay = [np.mean(delay_dict[i])*1000 for i in range(start,start+TIME_PERIOD)]
    min_delay = [min(delay_dict[i])*1000 for i in range(start,start+TIME_PERIOD)]
    max_delay = [max(delay_dict[i])*1000 for i in range(start,start+TIME_PERIOD)]
    nintey_delay = [sorted(delay_dict[i])[int(len(delay_dict[i])*0.9)]*1000 for i in range(start,start+TIME_PERIOD)]
    band_width = [band_width_dict[i]/1000 for i in range(start, start+TIME_PERIOD)]
    return mean_delay, min_delay, max_delay, nintey_delay, band_width

def plot_graph(x,y,name,ylabel):
    plt.figure()
    plt.plot(x,y) 
    plt.xlabel("time (s)")
    plt.ylabel(ylabel)
    plt.title(f'time vs {name}') 
    plt.savefig(f'{name}.png')

if __name__ == '__main__':
    mean_delay,min_delay, max_delay, nintey_delay, band_width = get_delay_bandwidth(logs_8,NODE8_START) 
    time = [i for i in range(1,TIME_PERIOD+1)]
    plot_graph(time, mean_delay,"mean delay", "mean delay (ms)")
    plot_graph(time, min_delay,"min delay", "min delay (ms)")
    plot_graph(time, max_delay,"max delay", "max delay (ms)")
    plot_graph(time, nintey_delay,"nintey delay", "nintey delay (ms)")
    plot_graph(time, band_width,"band width", "band width (kbp)")
    




        

