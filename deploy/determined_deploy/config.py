MASTER_IP = "localhost"
MASTER_PORT = "8080"


def make_master_url(suffix: str = "") -> str:
    return "http://{}:{}/{}".format(MASTER_IP, MASTER_PORT, suffix)
