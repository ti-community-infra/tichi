import Head from 'next/head'
import styles from '../styles/home.module.scss'
import { Button } from 'antd'

export default function Home() {
  return (
    <div className={styles.container}>
      <Head>
        <title>Owner Page</title>
      </Head>
      <div>hhh</div>
      <Button type="primary">Test Case</Button>
    </div>
  )
}
