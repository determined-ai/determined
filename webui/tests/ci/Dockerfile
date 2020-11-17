FROM node:12.19.1

# Install dependencies for headless chrome
# https://github.com/Googlechrome/puppeteer/issues/290#issuecomment-322838700
RUN apt-get update && apt-get install -q -y \
    apt-transport-https gnupg2 ca-certificates \
    git-all gconf-service libasound2 libatk1.0-0 libc6 libcairo2 libcups2 libdbus-1-3 \
    libexpat1 libfontconfig1 libgcc1 libgconf-2-4 libgdk-pixbuf2.0-0 libglib2.0-0 libgtk-3-0 \
    libnspr4 libpango-1.0-0 libpangocairo-1.0-0 libstdc++6 libx11-6 libx11-xcb1 libxcb1 libxcomposite1 \
    libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 libxss1 libxtst6 ca-certificates \
    fonts-liberation libappindicator1 libnss3 lsb-release xdg-utils wget && \
    rm -rf /var/lib/apt/lists/*

# Install gauge
RUN APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=1 apt-key adv --keyserver hkp://ipv4.pool.sks-keyservers.net --recv-keys 023EDB0B && \
    echo deb https://dl.bintray.com/gauge/gauge-deb stable main | tee -a /etc/apt/sources.list

RUN apt-get update && apt-get install gauge

# Install gauge plugins
RUN gauge install ts && \
    gauge install screenshot

ENV PATH=$HOME/.gauge:$PATH

CMD [ "gauge", "run", "--env", "ci", "specs" ]
