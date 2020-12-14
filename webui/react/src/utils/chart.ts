type Range = [number, number];

interface Config {
  batchSize: number;
  trialCount: number;
  stepCount: number;
  valueRange: Range;
}

type Point = { x: number; y: number };

const CHART_CONFIG: Config = {
  batchSize: 100,
  stepCount: 1000,
  trialCount: 100,
  valueRange: [ 0, 1 ],
};

export function generateSeries(steps?: number): Point[] {
  const maxSteps = steps !== undefined ? steps : CHART_CONFIG.stepCount;
  const series = [];
  const offset = (Math.random() * 2 - 1) * 2;

  for (let i = 0; i < maxSteps; i++) {
    const x = (i + 1) * CHART_CONFIG.batchSize;
    const variation = (Math.random() * 2 - 1) * 0.5;
    const y = 10 - Math.log(i + 1) + variation + offset;
    series.push({ x, y });
  }

  return series;
}

export function generateTrials(): Point[][] {
  return new Array(CHART_CONFIG.trialCount).fill(null).map(() => generateSeries());
}

export function generateScatter(
  count = 100,
  xRange: Range = [ 0, 1 ],
  yRange: Range = [ 0, 1 ],
): Point[] {
  return new Array(count).fill(null).map(() => {
    const x = Math.random() * (xRange[1] - xRange[0]) + xRange[0];
    const y = Math.random() * (yRange[1] - yRange[0]) + yRange[0];
    return { x, y };
  });
}

export function distance(x0: number, y0: number, x1: number, y1: number): number {
  return Math.sqrt((x1 - x0) ** 2 + (y1 - y0) ** 2);
}
