import Card from 'hew/Card';
import { Body, Label } from 'hew/Typography';
import React from 'react';

import css from './OverviewStats.module.scss';

interface Props {
  children: React.ReactNode;
  focused?: boolean;
  onClick?: () => void;
  title: string;
}

const OverviewStats: React.FC<Props> = (props: Props) => {
  const childClasses = [css.info];
  if (props.onClick) childClasses.push(css.clickable);

  return (
    <Card onClick={props.onClick}>
      <div className={css.base}>
        <div className={css.title}>
          <Label truncate={{ rows: 1, tooltip: true }}>{props.title}</Label>
        </div>
        <strong className={childClasses.join(' ')}>
          <Body truncate={{ rows: 1, tooltip: true }}>{props.children}</Body>
        </strong>
      </div>
    </Card>
  );
};

export default OverviewStats;
