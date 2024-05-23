import logging
import socket
from http.server import BaseHTTPRequestHandler, HTTPServer

import determined as det


class HelloHandler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        self.send_response(200)
        self.send_header("Content-type", "text/plain")
        self.end_headers()
        self.wfile.write(b"Hello")

    def do_POST(self) -> None:
        self.do_GET()


def start_http_server(
    server_class=HTTPServer, handler_class=HelloHandler, port: int = 8888
) -> None:
    server_address = ("", port)
    httpd = server_class(server_address, handler_class)
    print(f"Starting HTTP server on port {port}")
    httpd.serve_forever()


def start_server(host: str = "127.0.0.1", port: int = 8888) -> None:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind((host, port))
        s.listen()
        print(f"Server listening on {host}:{port}")

        while True:
            conn, addr = s.accept()
            with conn:
                print(f"Connected by {addr}")
                while True:
                    data = conn.recv(1024)
                    if not data:
                        break
                    try:
                        print(f"Received: {data.decode()}")
                    except UnicodeDecodeError:
                        print(f"Received: {data}")
                    conn.sendall(data)


def run():
    # start_server()
    start_http_server()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    if info is None:
        run()
        exit(0)
    slots_per_node = len(info.slot_ids)
    num_nodes = len(info.container_addrs)
    cross_rank = info.container_rank
    chief_ip = info.container_addrs[0]
    print(info)
    print(f"cross_rank: {cross_rank}, chief_ip: {chief_ip}")
    if cross_rank == 0:
        run()
    else:
        print("Not the chief, exiting.")
