import { Card } from "antd";
import { getReq } from "@/utils/request";
import { BASE_URL, COMMITTER, REVIEWER } from "@/utils/constant";
import style from "./owners.module.scss";
import { IOwnerTypeData } from "@/types/owners";

const { Meta } = Card;

export default function Owner({ data }) {
  const detailData: Partial<IOwnerTypeData> = data.data;
  const renderItem = (items: Array<string>) => {
    return items.map((item) => (
      <Card
        className={style.container}
        cover={<img alt="pic" src={`https://github.com/${item}.png`} />}
      >
        <a href={`https://github.com/${item}`}>{item}</a>
      </Card>
    ));
  };

  const reviewItems = detailData.reviewers
    ? renderItem(detailData.reviewers)
    : null;
  const committerItmes = detailData.committers
    ? renderItem(detailData.committers)
    : null;
  return (
    <>
      <p className={style.header}>needsLGTM: {detailData.needsLGTM}</p>
      {reviewItems ? (
        <>
          <p className={style.people}>{REVIEWER}</p>
          <div className={style.wrapper}>{reviewItems}</div>
        </>
      ) : null}
      {committerItmes ? (
        <>
          <p className={style.people}>{COMMITTER}</p>
          <div className={style.wrapper}>{committerItmes}</div>
        </>
      ) : null}
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
