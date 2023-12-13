import Card from 'hew/Card';
import Column from 'hew/Column';
import Row from 'hew/Row';
import { Body, Label, TypographySize } from 'hew/Typography';
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
      <Column>
        <Row>
          <Label size={TypographySize.XS} truncate={{ rows: 1, tooltip: true }}>
            {props.title}
          </Label>
        </Row>
        <Row>
          <strong className={childClasses.join(' ')}>
            <Body truncate={{ rows: 1, tooltip: true }}>{props.children}</Body>
          </strong>
        </Row>
      </Column>
    </Card>
  );
};

export default OverviewStats;
