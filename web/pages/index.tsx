import Head from "next/head";
import styles from "../styles/home.module.scss";
import { Button } from "antd";
import { useRouter } from "next/router";

export default function Owner() {
  const router = useRouter();
  return (
    <div className={styles.container}>
      <Head>
        <title>Owner Page</title>
      </Head>
      <div>Index Page</div>
    </div>
  );
}
