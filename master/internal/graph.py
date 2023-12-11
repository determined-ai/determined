import matplotlib.pyplot as plt

def parse_time(time_str):
    # Split the time string into value and unit
    value, unit = time_str[:-2], time_str[-2:]

    # Convert the value to milliseconds based on the unit
    if unit.lower() == 'ms':
        return float(value)
    elif unit.lower() == 's':
        return float(value) * 1000
    elif unit.lower() == 'µs':
        return float(value) / 1000
    elif unit.lower() == 'ns':
        return float(value) / 1000000
    else:
        print("UNIQT", unit)
        raise ValueError(f"Unsupported unit: {unit}")

def parse_file(file_path):
    '''
    data = {'isValidation': [], 'checkTrialRunID': [], 'rollbackMetrics': [],
            'summaryMetrics scan': [], 'addMetricsWithMerge': [], 'summary metrics create': [],
            'calculate new summary metrics': [], 'validations ID': [],
            'summary metrics debug check': [], 'update runs': [],
            'set best trial validation': [], 'ALL': []}
    '''
    data = {"epoch": []}
    with open(file_path, 'r') as file:
        for line in file:            
            if "PASS" in line or "ok" in line or "panic" in line or "running" in line or "TestSlowdown" in line:
                continue
            parts = line.split()
            
            print(parts)
            if "PASS:" in parts or 'migrations' in parts or 'while..."' in parts or "RUN" in parts or len(parts) == 0:
                continue

            if "epoch" in line:
                print(line.split("="))
                data["epoch"].append(int(line.split("=")[-1]))

            
            try:
                time = parse_time(parts[0])
            except:
                continue

            cur_graph = ' '.join(parts[1:])
            if cur_graph not in data:
                data[cur_graph] = []
            data[cur_graph].append(time)

    print(data)
    return data


def plot_graph(data):
    #labels = list(data.keys())
    #times = [item for sublist in data.values() for item in sublist]

    for l in data:
        if l == "epoch":
            continue
        plt.plot(data["epoch"], data[l], label=l)

    plt.xlabel('epoch / # metrics reported')
    plt.ylabel('Time (milliseconds)')
        
    plt.legend() 
    plt.show()    
    #plt.figure(figsize=(10, 6))
    #plt.barh(labels, times, color='skyblue')
    #plt.title('Execution Time for Each Operation')
    #plt.show()

if __name__ == "__main__":
    file_path = "out.log"  # Replace with the actual path to your file
    execution_data = parse_file(file_path)
    plot_graph(execution_data)
