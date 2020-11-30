import Head from 'next/head'
import styles from '../styles/home.module.scss'

export default function Home() {
  return (
    <div className={styles.container}>
      <Head>
        <title>Owner Page</title>
      </Head>
    </div>
  )
}
