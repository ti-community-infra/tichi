import Head from "next/head";

import styles from "../styles/home.module.scss";
import React from "react";

export default function Owner() {
  return (
    <div className={styles.container}>
      <Head>
        <title>Ti Chi</title>
      </Head>

      <div>Ti Chi</div>
    </div>
  );
}
