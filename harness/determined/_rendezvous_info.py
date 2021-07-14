from typing import List


class RendezvousInfo:
    """
    RendezvousInfo was machine identity information that is:
     - configured as an environment variable in the rendezvous layer
     - independent of the launch layer
     - consumed by the launch layer
    """

    def __init__(self, addrs: List[str], rank: int):
        self.addrs = addrs
        self.rank = rank

    def get_rank(self) -> int:
        return self.rank

    def get_size(self) -> int:
        return len(self.addrs)

    def get_addrs(self) -> List[str]:
        """
        Returns the addresses of all the gang members.
        """

        return self.addrs

    def get_ip_addresses(self) -> List[str]:
        """
        Returns the ip addresses of all the gang members.
        """

        return [addr.split(":")[0] for addr in self.addrs]

    def get_ports(self) -> List[int]:
        """
        Returns the first port address of all gang members.
        """

        return [int(addr.split(":")[1]) for addr in self.addrs]
