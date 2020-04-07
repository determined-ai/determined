from typing import List


class RendezvousInfo:
    def __init__(self, addrs: List[str], addrs2: List[str], rank: int):
        self.addrs = addrs
        self.addrs2 = addrs2
        self.rank = rank

    def get_rank(self) -> int:
        """
        Returns the distributed training rank information.
        """

        return self.rank

    def get_size(self) -> int:
        """
        Returns the number of distributed training machines.
        """

        return len(self.addrs)

    def get_addrs(self) -> List[str]:
        """
        Returns the addresses of all the gang members.
        """

        return self.addrs

    def get_addrs2(self) -> List[str]:
        """
        Returns the addresses of all the gang members.
        """

        return self.addrs2

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

    def get_second_ports(self) -> List[int]:
        """
        Returns the second port address of all gang members.
        """

        return [int(addr.split(":")[1]) for addr in self.addrs2]

    def is_distributed_trial(self) -> bool:
        return len(self.get_addrs()) > 1
