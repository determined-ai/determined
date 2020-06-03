import { Typography } from 'antd';
import React from 'react';

import { CommonProps } from 'types';
import { toHtmlId } from 'utils/string';

import css from './Page.module.scss';

const { Title } = Typography;

interface Props extends CommonProps {
  title: string;
  hideTitle?: boolean;
}

const defaultProps = {
  hideTitle: false,
};

const Page: React.FC<Props> = (props: Props) => {
  const classes = [ props.className, css.base ];

  return (
    <main className={classes.join(' ')} id={toHtmlId(props.title)}>
      {props.hideTitle || <Title className={css.title}>{props.title}</Title>}
      <div className={css.body}>
        {props.children}
      </div>
    </main>
  );
};

Page.defaultProps = defaultProps;

export default Page;
