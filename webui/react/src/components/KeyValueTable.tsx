import React from 'react';

interface TableProps {
  boldKeys?: boolean;
  header?: React.ReactNode;
  lines: Line[];
}

interface Line {
  key: string;
  value: string | string[]
}

interface LineProps {
  line: Line;
}

const KeyValueTable: React.FC<TableProps> = ({ header, lines }: TableProps) => {
  return (
    <div>
      {header != null ? <div>{header}</div>: null}
      <div>
        {lines.map(line => <Line key={line.key} line={line} />)}
      </div>
    </div>
  );
};

const Line: React.FC<LineProps> = ({ line }: LineProps) => {
  return (
    <>
      <p>{line.key}</p>
      {Array.isArray(line.value) ?
        <div>{line.value.map((val, i) => <p key={i}>{val}</p>)}</div> :
        <p>{line.value}</p>}
    </>
  );
};

export default KeyValueTable;
