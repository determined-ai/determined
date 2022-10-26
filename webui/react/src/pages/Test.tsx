import React, { useState } from 'react';
import { useRecoilState } from 'recoil';

import { config } from 'useWebSettings/useWebSettings.settings';

const Test: React.FC = () => {
  const [count, setCount] = useState<number>(0);
  const [tableLimit, setTableLimit] = useRecoilState<number>(config.settings.tableLimit.atom);
  // const [columns, setColumns] = useRecoilState<string[]>(config.settings.columns.atom);
  // const [columnWidths, setColumnWidths] = useRecoilState<number[]>(
  //   config.settings.columnWidths.atom,
  // );
  const [numOfCake, setNumOfCake] = useRecoilState<number>(config.settings.numOfCake.atom);
  const [letter, setLetter] = useRecoilState<string>(config.settings.letter.atom);

  const onClick = () => {
    setTableLimit((prev) => prev + 1);
    // setColumns([]);
    // setColumnWidths([count]);
    setNumOfCake((prev) => prev + 1);
    setLetter(`${numOfCake}`);
    setCount((prev) => prev + 1);
  };

  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>tableLimit: {tableLimit}</div>
      <div>numOfCake: {numOfCake}</div>
      <div>letter: {letter}</div>
      {/* <div>columns: {JSON.stringify(columns)}</div> */}
      {/* <div>columnWidths: {JSON.stringify(columnWidths)}</div> */}
    </>
  );
};

export default Test;
