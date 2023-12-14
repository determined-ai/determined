import Card from 'hew/Card';
import Column from 'hew/Column';
import Link from 'hew/Link';
import Row from 'hew/Row';
import { Body, Label, TypographySize } from 'hew/Typography';
import React from 'react';

interface Props {
  children: React.ReactNode;
  focused?: boolean;
  onClick?: () => void;
  title: string;
}

const OverviewStats: React.FC<Props> = (props: Props) => {
  const column = (
    <Column>
      <Row>
        <Label size={TypographySize.XS} truncate={{ rows: 1, tooltip: true }}>
          {props.title}
        </Label>
      </Row>
      <Row>
        <strong>
          <Body truncate={{ rows: 1, tooltip: true }}>{props.children}</Body>
        </strong>
      </Row>
    </Column>
  );
  return <Card>{props.onClick ? <Link onClick={props.onClick}>{column}</Link> : column}</Card>;
};

export default OverviewStats;
