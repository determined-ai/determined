import { useEffect, useMemo, useState } from 'react';

import { glasbeyColor } from 'utils/color';

export type MapOfIdsToColors = Record<string, string>;

export const useGlasbey = (ids: string[] | number[]): MapOfIdsToColors => {
  const itemIds = useMemo(() => ids.map(String), [ids]);
  const [indexMap, setIndexMap] = useState(() =>
    itemIds.map((itemId, idx) => ({ [itemId]: idx })).reduce((a, b) => ({ ...a, ...b }), {}),
  );

  useEffect(() => {
    const idsInMap = Object.keys(indexMap);
    const idsToAdd = itemIds.filter((id) => !idsInMap.includes(id));
    const idsToRemove = idsInMap.filter((id) => !itemIds.includes(id));
    if (idsToAdd.length === 0 && idsToRemove.length === 0) return;
    const newIndexMap = Object.entries(indexMap)
      .map(([id, glasbeyIndex]) => (itemIds.includes(id) ? { [id]: glasbeyIndex } : {}))
      .reduce((a, b) => ({ ...a, ...b }), {});

    let tryIndex = 0;
    while (idsToAdd.length) {
      const currentId = idsToAdd.shift();
      const takenIndices = Object.values(newIndexMap);
      if (currentId === undefined) continue;
      while (takenIndices.includes(tryIndex)) tryIndex += 1;
      newIndexMap[currentId] = tryIndex;
    }

    setIndexMap(newIndexMap);
  }, [itemIds, indexMap]);

  const colorMap = useMemo(
    () =>
      Object.entries(indexMap)
        .map(([id, glasbeyIndex]) => ({ [id]: glasbeyColor(glasbeyIndex) }))
        .reduce((a, b) => ({ ...a, ...b }), {}),
    [indexMap],
  );

  return colorMap;
};
