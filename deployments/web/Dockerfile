FROM node:12

# Create app directory
RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

# Bundle app source
COPY web /usr/src/app

# Install app dependencies
RUN npm ci

# Build app
RUN npm run build

# This run the server at default port 3000
EXPOSE 3000
CMD ["npm", "run", "start"]