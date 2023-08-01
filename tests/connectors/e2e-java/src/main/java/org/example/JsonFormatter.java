package org.example;

import com.google.common.collect.Maps;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.annotations.SerializedName;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.Map;
import java.util.Set;
import java.util.logging.Formatter;
import java.util.logging.Level;
import java.util.logging.LogRecord;

/**
 * Produces structured logs for GKE in accordance with
 * https://cloud.google.com/logging/docs/structured-logging
 */
public class JsonFormatter extends Formatter {
  private Map<String,String> labels;

  @Override
  public String format(LogRecord record) {
    GsonBuilder b = new GsonBuilder();
    Gson gson = b.create();
    String s = gson.toJson(new StructuredLogRecord(
        record.getInstant(), severityFor(record.getLevel()), record.getMessage(), record.getThrown(),
        Map.of("log", record.getLoggerName(),
            "thread", threadName(record.getThreadID()),
            "class", record.getSourceClassName(),
            "method", record.getSourceMethodName()))) + "\n";
    return s;
  }

  private String threadName(long threadID) {
    Set<Thread> threadSet = Thread.getAllStackTraces().keySet();
    for (Thread t :threadSet) {
      if(t.getId() == threadID) {
        return t.getName();
      }
    }
    return "unknown";
  }

  private static class StructuredLogRecord {
    private final String severity;
    private final String message;
    @SerializedName("logging.googleapis.com/labels")
    private final Map<String,String> labels;
    private final Map<String,Long> timestamp;

    private StructuredLogRecord(Instant time, String severity, String message, Throwable thrown, Map<String, String> labels) {
      this.severity = severity;
      this.labels = labels;
      this.timestamp = Map.of(
          "seconds", Long.valueOf(time.getEpochSecond()),
          "nanos", Long.valueOf(time.getNano()));

      if(thrown != null) {
        StringWriter st = new StringWriter();
        PrintWriter w = new PrintWriter(st);
        w.println(message);
        thrown.printStackTrace(w);
        w.flush();
        this.message = st.toString();
      } else {
        this.message = message;
      }
    }

    public String getSeverity() {
      return severity;
    }

    public String getMessage() {
      return message;
    }


    public Map<String, String> getLabels() {
      return labels;
    }

    public Map<String, Long> getTimestamp() {
      return timestamp;
    }
  }
  private String severityFor(Level level) {
    if(level == null) {
      return "DEFAULT"; //	(0) The log entry has no assigned severity level.
    }
    if(level.intValue() <= Level.FINER.intValue()) {
      return "DEBUG"; //	(100) Debug or trace information.
    }
    if(level.intValue() <= Level.INFO.intValue()) {
      return "INFO"; //	(200) Routine information, such as ongoing status or performance.
    }
    if(level.intValue() <= Level.WARNING.intValue()) {
        return "WARNING"; //	(400) Warning events might cause problems.
    }
    if(level.intValue() <= Level.SEVERE.intValue()) {
      return "ERROR"; //	(500) Error events are likely to cause problems.
    }
    return "DEFAULT";
  }
}
