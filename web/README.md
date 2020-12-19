## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

## Server API

- [doc](https://book.prow.tidb.io/plugins/owners.html)

The api design is just like `/repos/:org/:repo/pulls/:numbers/owners`.

The api such as `https://prow.tidb.io/ti-community-owners/repos/pingcap/tidb-operator/pulls/3522/owners`

## How to visit

Page route just like: `/:org/:repo/pulls/:num/owners`
