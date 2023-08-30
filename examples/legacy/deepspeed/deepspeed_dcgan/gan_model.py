from typing import cast

import torch
import torch.nn as nn


def weights_init(m: nn.Module) -> None:
    classname = m.__class__.__name__
    if classname.find("Conv") != -1:
        nn.init.normal_(cast(torch.Tensor, m.weight.data), 0.0, 0.02)
    elif classname.find("BatchNorm") != -1:
        nn.init.normal_(cast(torch.Tensor, m.weight.data), 1.0, 0.02)
        nn.init.constant_(cast(torch.Tensor, m.bias.data), 0)


class Generator(nn.Module):
    def __init__(self, ngf: int, nc: int, nz: int) -> None:
        super(Generator, self).__init__()  # type: ignore
        self.main = nn.Sequential(
            # input is Z, going into a convolution
            nn.ConvTranspose2d(nz, ngf * 8, 4, 1, 0, bias=False),
            nn.BatchNorm2d(ngf * 8),  # type: ignore
            nn.ReLU(True),
            # state size. (ngf*8) x 4 x 4
            nn.ConvTranspose2d(ngf * 8, ngf * 4, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ngf * 4),  # type: ignore
            nn.ReLU(True),
            # state size. (ngf*4) x 8 x 8
            nn.ConvTranspose2d(ngf * 4, ngf * 2, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ngf * 2),  # type: ignore
            nn.ReLU(True),
            # state size. (ngf*2) x 16 x 16
            nn.ConvTranspose2d(ngf * 2, ngf, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ngf),  # type: ignore
            nn.ReLU(True),
            # state size. (ngf) x 32 x 32
            nn.ConvTranspose2d(ngf, nc, 4, 2, 1, bias=False),
            nn.Tanh()  # type: ignore
            # state size. (nc) x 64 x 64
        )

    def forward(self, input: torch.Tensor) -> torch.Tensor:
        output = self.main(input)
        return cast(torch.Tensor, output)


class Discriminator(nn.Module):
    def __init__(self, ndf: int, nc: int) -> None:
        super(Discriminator, self).__init__()  # type: ignore
        self.main = nn.Sequential(
            # input is (nc) x 64 x 64
            nn.Conv2d(nc, ndf, 4, 2, 1, bias=False),
            nn.LeakyReLU(0.2, inplace=True),
            # state size. (ndf) x 32 x 32
            nn.Conv2d(ndf, ndf * 2, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ndf * 2),  # type: ignore
            nn.LeakyReLU(0.2, inplace=True),
            # state size. (ndf*2) x 16 x 16
            nn.Conv2d(ndf * 2, ndf * 4, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ndf * 4),  # type: ignore
            nn.LeakyReLU(0.2, inplace=True),
            # state size. (ndf*4) x 8 x 8
            nn.Conv2d(ndf * 4, ndf * 8, 4, 2, 1, bias=False),
            nn.BatchNorm2d(ndf * 8),  # type: ignore
            nn.LeakyReLU(0.2, inplace=True),
            # state size. (ndf*8) x 4 x 4
            nn.Conv2d(ndf * 8, 1, 4, 1, 0, bias=False),
            nn.Sigmoid(),  # type: ignore
        )

    def forward(self, input: torch.Tensor) -> torch.Tensor:
        output = self.main(input)
        return cast(torch.Tensor, output.view(-1, 1).squeeze(1))
