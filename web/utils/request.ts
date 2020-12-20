import axios from 'axios';

export const getReq = (url: string): Promise<Array<Object>> => axios.get(url).then(res => res.data);
