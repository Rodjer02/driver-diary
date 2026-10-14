import { Alert, Check, Copy } from "./icons";

export type FeedbackState =
  | { kind: "created" }
  | { kind: "duplicate" }
  | { kind: "errors"; list: string[] }
  | null;

export default function Feedback({ state }: { state: FeedbackState }) {
  if (!state) return null;
  if (state.kind === "created") {
    return (
      <div role="status" className="feedback ok">
        <Check />
        <div>
          <span className="feedback-title">Поездка добавлена</span>
          <span className="feedback-sub">Итоги обновлены, форма очищена</span>
        </div>
      </div>
    );
  }
  if (state.kind === "duplicate") {
    return (
      <div role="status" className="feedback dup">
        <Copy />
        <div>
          <span className="feedback-title">Такая поездка уже есть — дубль не создан</span>
          <span className="feedback-sub">Итоги дня не изменились</span>
        </div>
      </div>
    );
  }
  return (
    <div role="alert" className="feedback err">
      <Alert />
      <div>
        {state.list.map((e) => (
          <span key={e} className="feedback-title">
            {e}
          </span>
        ))}
      </div>
    </div>
  );
}
