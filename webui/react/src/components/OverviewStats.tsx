import { Typography } from 'antd';
import React from 'react';

import Card from 'components/kit/Card';
import css from 'components/OverviewStats.module.scss';

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
        <Typography.Title className={css.title} ellipsis={{ rows: 1, tooltip: true }} level={5}>
          {props.title}
        </Typography.Title>
        <strong className={childClasses.join(' ')}>
          <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
            {props.children}
          </Typography.Paragraph>
        </strong>
      </div>
    </Card>
  );
};

export default OverviewStats;
