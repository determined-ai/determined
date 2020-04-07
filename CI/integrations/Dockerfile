FROM python:3.6.9

# Ensure that Python outputs everything from the application rather than
# buffering it.
ENV PYTHONUNBUFFERED 1
ENV VIRTUAL_ENV 1

ENV NVIDIA_VISIBLE_DEVICES all
ENV NVIDIA_DRIVER_CAPABILITIES compute,utility

COPY ./ ./
RUN pip install --no-cache-dir -r combined-reqs.txt

RUN rm -rf /usr/local/go; curl -fsSL https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"
