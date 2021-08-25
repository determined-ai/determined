import React from 'react';

import css from './InfoBox.module.scss';

export interface InfoRow {
  content?: React.ReactNode | React.ReactNode[];
  label: string;
}

interface Props {
  header?: React.ReactNode;
  rows: InfoRow[];
}

export const renderRow = ({ label, content }: InfoRow): React.ReactNode => {
  if (content == null) return null;
  return (
    <div className={css.info} key={label}>
      <div className={css.label}>{label}</div>
      {Array.isArray(content) ?
        <div className={css.contentList}>
          {content.map((item, idx) => <div className={css.content} key={idx}>{item}</div>)}
        </div> :
        <div className={css.content}>{content}</div>
      }

    </div>
  );
};

const InfoBox: React.FC<Props> = ({ header, rows }: Props) => {
  return (
    <div className={css.base}>
      {header != null ? <div className={css.header}>{header}</div>: null}
      {rows.map(renderRow)}
    </div>
  );
};

export default InfoBox;
