import { Card } from "antd";
import { getReq } from "@/utils/request";
import { BASE_URL, COMMITTER, REVIEWER } from "@/utils/constant";
import style from "./owners.module.scss";
import { IOwnerTypeData } from "@/types/owners";

export default function Owner({ data }) {
  const detailData: Partial<IOwnerTypeData> = data.data;
  const committerItmes = detailData.committers
    ? detailData.committers.map((item) => (
        <p key={item} className={style.items}>
          {item}
        </p>
      ))
    : null;
  const reviewItems = detailData.reviewers
    ? detailData.reviewers.map((item) => (
        <p key={item} className={style.items}>
          {item}
        </p>
      ))
    : null;
  const renderItem = (item: Array<React.ReactElement> | null, key: string) => {
    return item ? (
      <Card title={key} className={style.container}>
        {item}
      </Card>
    ) : null;
  };
  return (
    <>
      <p className={style.header}>needsLGTM: {detailData.needsLGTM}</p>
      <div className={style.wrapper}>
        {renderItem(committerItmes, COMMITTER)}
        {renderItem(reviewItems, REVIEWER)}
      </div>
    </>
  );
}

export async function getServerSideProps(ctx) {
  const { query } = ctx;
  const { num, org, repo } = query;
  let data;
  try {
    data = await getReq(`${BASE_URL}/${org}/${repo}/pulls/${num}/owners`);
  } catch (err) {
    throw err;
  }
  return {
    props: { data },
  };
}
