import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);

export const formatDatetime = (datetime: string, format?: string, utc = true): string => {
  // Read as UTC or local time.
  const datetimeObject = utc ? dayjs.utc(datetime) : dayjs(datetime);
  return datetimeObject.format(format || 'YYYY-MM-DD, HH:mm:ss');
};
