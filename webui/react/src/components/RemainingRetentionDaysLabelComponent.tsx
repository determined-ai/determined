import React, { useMemo } from 'react';

import { pluralizer } from 'utils/string';

import Badge from './Badge';

interface Props {
  remainingLogDays: number | undefined;
}

const RemainingRetentionDaysLabel: React.FC<Props> = ({ remainingLogDays }: Props) => {
  const RemainingRetentionDaysLabelComponent = useMemo(() => {
    let toolTipText = '';
    let badgeText = '';
    if (remainingLogDays === undefined) {
      toolTipText = 'Days remaining to retention are not available yet.';
      badgeText = '-';
    } else if (remainingLogDays === -1) {
      toolTipText = 'Logs will be retained forever.';
      badgeText = 'Frvr';
    } else if (remainingLogDays === 0) {
      toolTipText = 'Some logs have begun to be deleted for this trial.';
      badgeText = '0';
    } else {
      toolTipText = `${remainingLogDays} ${pluralizer(
        remainingLogDays,
        'day',
      )} left to retain logs`;
      badgeText = `${remainingLogDays}`;
    }
    return (
      <div>
        <span>Logs </span>
        <Badge tooltip={toolTipText}>{badgeText}</Badge>
      </div>
    );
  }, [remainingLogDays]);

  return RemainingRetentionDaysLabelComponent;
};

export default RemainingRetentionDaysLabel;
