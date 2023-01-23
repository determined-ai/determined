import { observable } from 'micro-observables';
import React, { createContext, useContext, useMemo, useState } from 'react';
import uPlot, { AlignedData } from 'uplot';

type Bounds = {
  dataBounds: {
    max: number;
    min: number;
  } | null;
  zoomBounds: {
    max: number;
    min: number;
  } | null;
};

class SyncService {
  bounds = observable<Bounds>({ dataBounds: null, zoomBounds: null });

  pubSub: uPlot.SyncPubSub;

  activeBounds = this.bounds.select((b) => b?.zoomBounds ?? b?.dataBounds);

  constructor(pubSub: uPlot.SyncPubSub) {
    this.pubSub = pubSub;
    this.activeBounds.subscribe((b) => {
      if (!b) return;
      pubSub.plots.forEach((u) => {
        u.setScale('x', { max: b.max, min: b.min });
      });
    });
  }

  resetZoom() {
    this.bounds.update((b) => ({ ...b, zoomBounds: null }));
  }

  setZoom(min: number, max: number) {
    this.bounds.update((b) => ({ ...b, zoomBounds: { max, min } }));
  }

  updateDataBounds(data: AlignedData) {
    const xValues = data[0];
    const lastIdx = xValues.length - 1;
    const dataMin = xValues[0];
    const dataMax = xValues[lastIdx];
    this.bounds.update((b) => {
      const newMax = Math.max(b.dataBounds?.max ?? dataMax, dataMax);
      const newMin = Math.min(b.dataBounds?.min ?? dataMin, dataMin);
      return { ...b, dataBounds: { max: newMax, min: newMin } };
    });
  }
}

interface Props {
  children: React.ReactNode;
}

const SyncContext = createContext<SyncService | null>(null);

export const SyncProvider: React.FC<Props> = ({ children }) => {
  const [syncService] = useState(() => new SyncService(uPlot.sync('x')));

  return <SyncContext.Provider value={syncService}>{children}</SyncContext.Provider>;
};

export const useChartSync = (): {
  options: Partial<uPlot.Options>;
  syncService: SyncService;
} => {
  const syncService = useContext(SyncContext);

  const [dummyService] = useState(() => new SyncService(uPlot.sync('x')));

  const options = useMemo(() => {
    const syncKey = syncService?.pubSub.key;
    return {
      cursor: {
        bind: {
          dblclick: () => {
            return () => {
              syncService?.resetZoom();
              return null;
            };
          },
        },
        drag: { dist: 5, setScale: false, uni: 10, x: true, y: false },
        sync: {
          key: syncKey,
          scales: [syncKey, null],
          setSeries: false,
        },
      },
      hooks: {
        ready: [
          (chart: uPlot) => {
            const activeBounds = syncService?.activeBounds.get();
            if (activeBounds) chart.setScale('x', activeBounds);
          },
        ],
        setSelect: [
          (chart: uPlot) => {
            const min = chart.posToVal(chart.select.left, 'x');
            const max = chart.posToVal(chart.select.left + chart.select.width, 'x');
            syncService?.setZoom(min, max);
            chart.setSelect({ height: 0, left: 0, top: 0, width: 0 }, false);
          },
        ],
      },
    };
  }, [syncService]) as Partial<uPlot.Options>;

  return syncService
    ? { options, syncService }
    : {
        options: {
          cursor: {
            drag: { dist: 5, uni: 10, x: true, y: true },
          },
        },
        syncService: dummyService,
      };
};
