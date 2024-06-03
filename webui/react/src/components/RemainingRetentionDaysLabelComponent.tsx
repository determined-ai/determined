import React, { useMemo } from 'react';

import { pluralizer } from 'utils/string';

import Badge from './Badge';

interface Props {
  remainingLogDays: number | undefined;
}

const RemainingRetentionDaysLabel: React.FC<Props> = ({ remainingLogDays }: Props) => {
  const [toolTipText, badgeText] = useMemo(() => {
    let toolTipTxt = '';
    let badgeTxt = '';
    if (remainingLogDays === undefined) {
      toolTipTxt = 'Days remaining to retention are not available yet.';
      badgeTxt = '-';
    } else if (remainingLogDays === -1) {
      toolTipTxt = 'Logs will be retained forever.';
      badgeTxt = 'Frvr';
    } else if (remainingLogDays === 0) {
      toolTipTxt = 'Some logs have begun to be deleted for this trial.';
      badgeTxt = '0';
    } else {
      toolTipTxt = `${remainingLogDays} ${pluralizer(remainingLogDays, 'day')} left to retain logs`;
      badgeTxt = `${remainingLogDays}`;
    }
    return [toolTipTxt, badgeTxt];
  }, [remainingLogDays]);

  return (
    <div>
      <span>Logs </span>
      <Badge tooltip={toolTipText}>{badgeText}</Badge>
    </div>
  );
};

export default RemainingRetentionDaysLabel;
