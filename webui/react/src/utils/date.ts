import dayjs from 'dayjs';

export const formatDatetime = (datetime: string, format?: string): string => {
  return dayjs(datetime).format(format || 'YYYY-MM-DD HH:mm:ss');
};
