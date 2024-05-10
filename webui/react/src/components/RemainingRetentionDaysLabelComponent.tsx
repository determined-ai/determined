import { Loadable } from 'hew/utils/loadable';
import React from 'react';

import { pluralizer } from 'utils/string';

import Badge from './Badge';

interface Props {
  remainingLogDays: Loadable<number | undefined>;
}

const RemainingRetentionDaysLabelComponent: React.FC<Props> = ({ remainingLogDays }: Props) => {
  let toolTipText = '';
  let badgeText = '';
  const days = Loadable.getOrElse(undefined, remainingLogDays);
  if (days === undefined) {
    toolTipText = 'Days remaining to retention are not available yet.';
    badgeText = '-';
  } else if (days === -1) {
    toolTipText = 'Logs will be retained forever.';
    badgeText = 'Frvr';
  } else if (days === 0) {
    toolTipText = 'Some logs have begun to be deleted for this trial.';
    badgeText = '0';
  } else {
    toolTipText = `${days} ${pluralizer(days, 'day')} left to retain logs`;
    badgeText = `${days}`;
  }
  return (
    <div>
      <span>Logs </span>
      <Badge tooltip={toolTipText}>{badgeText}</Badge>
    </div>
  );
};

export default RemainingRetentionDaysLabelComponent;
