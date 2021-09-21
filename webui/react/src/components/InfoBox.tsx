import React from 'react';

import css from './InfoBox.module.scss';

export interface InfoRow {
  content?: React.ReactNode | React.ReactNode[];
  label: string;
}

interface InfoRowProps extends InfoRow {
  seperator: boolean;
}

interface Props {
  header?: React.ReactNode;
  rows: InfoRow[];
  seperator?: boolean;
}

export const renderRow = ({ label, content, seperator }: InfoRowProps): React.ReactNode => {
  if (content == null) return null;
  return (
    <div className={[ css.info, seperator ? css.seperator : null ].join(' ')} key={label}>
      <dt className={css.label}>{label}</dt>
      {Array.isArray(content) ?
        <dd className={css.contentList}>
          {content.map((item, idx) => <div className={css.content} key={idx}>{item}</div>)}
        </dd> :
        <dd className={css.content}>{content}</dd>
      }

    </div>
  );
};

const InfoBox: React.FC<Props> = ({ header, rows, seperator = true }: Props) => {
  return (
    <dl className={css.base}>
      {header != null ? <div className={css.header}>{header}</div>: null}
      {rows.map((row) => renderRow({ ...row, seperator }))}
    </dl>
  );
};

export default InfoBox;
