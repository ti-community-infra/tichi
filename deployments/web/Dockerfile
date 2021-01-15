FROM node:12

ARG NODE_ENV="production"
ENV NODE_ENV=${NODE_ENV}

# Create app directory
RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

# Bundle app source
COPY web /usr/src/app

# Install app dependencies
RUN npm install

# Build app
RUN npm run build

# This run the server at default port 3000
EXPOSE 3000
CMD ["npm", "run", "start"]