/** The screens of the setup guide, in order. The folders screen is only visited when file search is on. */
export type Step = "welcome" | "hotkey" | "look" | "features" | "files" | "done";

const BASE: Step[] = ["welcome", "hotkey", "look", "features"];

/** The screens to walk through, given whether file search is on. */
export function stepsFor(files: boolean): Step[] {
  return files ? [...BASE, "files", "done"] : [...BASE, "done"];
}

/** The screen after `step`, or the same one at the end. */
export function after(step: Step, files: boolean): Step {
  const steps = stepsFor(files);
  return steps[Math.min(steps.indexOf(step) + 1, steps.length - 1)] ?? "done";
}

/** The screen before `step`, or the same one at the start. */
export function before(step: Step, files: boolean): Step {
  const steps = stepsFor(files);
  return steps[Math.max(steps.indexOf(step) - 1, 0)] ?? "welcome";
}
