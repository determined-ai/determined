import json

def main():
    with open('logs.log', 'r') as f:
        lines = f.readlines()

    current_measurement = None
    last_time = None
    for line in lines:
        if "SPLICE_HERE" not in line:
            continue

        datapoint_str = line.split("SPLICE_HERE")[1].strip().replace("'", '"')
        j = json.loads(datapoint_str)
        # print(datapoint_json)

        if j["measurement"] != current_measurement:
            current_measurement = j["measurement"]
            last_time = j["timestamp"]
            continue

        time_elapsed_since_last_measurement = j["timestamp"] - last_time
        last_time = j["timestamp"]
        human_readable_elapsed = str(time_elapsed_since_last_measurement * 1000) + " ms"

        print(j["measurement"], time_elapsed_since_last_measurement)




if __name__ == '__main__':
    main()