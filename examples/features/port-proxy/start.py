import logging
import socket

import determined as det


def start_server(host: str = "127.0.0.1", port: int = 8888) -> None:
    # create a socket object
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        # bind the socket to the host and port
        s.bind((host, port))
        # listen for incoming connections
        s.listen()
        print(f"Server listening on {host}:{port}")

        while True:
            # accept a new connection
            conn, addr = s.accept()
            with conn:
                print(f"Connected by {addr}")
                while True:
                    data = conn.recv(1024)  # receive data from client
                    if not data:
                        break
                    print(f"Received: {data.decode()}")
                    conn.sendall(data)  # echo the data back to the client


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    assert info is not None, "this example only runs on-cluster"
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank
    chief_ip = info.container_addrs[0]
    print(info)
    print(f"cross_rank: {cross_rank}, chief_ip: {chief_ip}")
    if cross_rank == 0:
        start_server()
    else:
        print("Not the chief, exiting.")
