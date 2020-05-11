import argparse
import socket
import time


def wait_for_server(host, port, timeout=5.0):
    for _ in range(100):
        try:
            with socket.create_connection((host, port), timeout=timeout):
                return
        except OSError:
            time.sleep(1)
    raise Exception(f"Timed out waiting for the {host}:{port}.")


def main() -> None:
    parser = argparse.ArgumentParser(description="Wait for server helper.")
    parser.add_argument("host", help="Host")
    parser.add_argument("port", help="Port")
    args = parser.parse_args()
    wait_for_server(args.host, args.port)


if __name__ == "__main__":
    main()
