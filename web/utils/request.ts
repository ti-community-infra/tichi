import axios from 'axios';

export const get = (url: string): Promise<Array<Object>> => axios.get(url).then(res => res.data);
