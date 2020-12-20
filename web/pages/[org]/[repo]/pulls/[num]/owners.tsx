import { Card } from "antd";
import { getReq } from "@/utils/request";
import { BASE_URL, COMMITTER, REVIEWER } from "@/utils/constant";
import style from "./owners.module.scss";
import { IOwnerTypeData } from "@/types/owners";

export default function Owner({ data }) {
  const detailData: Partial<IOwnerTypeData> = data.data;
  const renderItem = (items: Array<string>, title: string) => {
    return (
      <>
        <p className={style.people}>{title}</p>
        <div className={style.wrapper}>
          {items.map((item) => (
            <Card
              className={style.container}
              cover={<img alt="pic" src={`https://github.com/${item}.png`} />}
            >
              <a href={`https://github.com/${item}`}>{item}</a>
            </Card>
          ))}
        </div>
      </>
    );
  };

  const reviewItems = detailData.reviewers
    ? renderItem(detailData.reviewers, REVIEWER)
    : null;
  const committerItmes = detailData.committers
    ? renderItem(detailData.committers, COMMITTER)
    : null;
  return (
    <>
      <p className={style.header}>needsLGTM: {detailData.needsLGTM}</p>
      {reviewItems}
      {committerItmes}
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
