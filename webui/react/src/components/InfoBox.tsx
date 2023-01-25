import React from 'react';

import css from './InfoBox.module.scss';

export interface InfoRow {
  content?: React.ReactNode | React.ReactNode[];
  label: React.ReactNode;
}

interface InfoRowProps extends InfoRow {
  separator: boolean;
}

interface Props {
  header?: React.ReactNode;
  rows: InfoRow[];
  separator?: boolean;
}

export const renderRow = ({ label, content, separator }: InfoRowProps): React.ReactNode => {
  if (content === undefined) return null;
  return (
    <div className={[css.info, separator ? css.separator : null].join(' ')} key={label?.toString()}>
      <dt className={css.label}>{label}</dt>
      {Array.isArray(content) ? (
        <dd className={css.contentList}>
          {content.map((item, idx) => (
            <div className={css.content} key={idx}>
              {item}
            </div>
          ))}
        </dd>
      ) : (
        <dd className={css.content}>
          {content === null || String(content).trim() === '' ? (
            <code className={css.blank}> {content}</code>
          ) : (
            content
          )}
        </dd>
      )}
    </div>
  );
};

const InfoBox: React.FC<Props> = ({ header, rows, separator = true }: Props) => {
  return (
    <dl className={css.base}>
      {header != null ? <div className={css.header}>{header}</div> : null}
      {rows.map((row) => renderRow({ ...row, separator }))}
    </dl>
  );
};

export default InfoBox;
