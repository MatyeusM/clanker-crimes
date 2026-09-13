package com.clankerprise.sorting.persistence;

import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.domain.TagRepository;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Persists a {@link TagRepository} to {@codePointValue .index.json} inside the library folder.
 *
 * <p>Uses a small hand-rolled JSON reader/writer so the project needs zero third-party dependencies
 * and works offline on every platform.
 */
public final class JsonFileIndexRepository {

  public static final String INDEX_STORAGE_FILENAME = ".index.json";
  public static final String CONTENT_STORE_DIRECTORY_NAME = ".files";
  public static final int SCHEMA_VERSION = 1;

  private JsonFileIndexRepository() {}

  public static Path resolveIndexFile(Path libraryRoot) {
    return libraryRoot.resolve(INDEX_STORAGE_FILENAME);
  }

  public static Path resolveContentStoreDirectory(Path libraryRoot) {
    return libraryRoot.resolve(CONTENT_STORE_DIRECTORY_NAME);
  }

  /** Loads the index; returns an empty sourceRepository if none exists yet. */
  public static TagRepository hydrateRepository(Path libraryRoot) throws IOException {
    TagRepository sourceRepository = new TagRepository(libraryRoot);
    Path indexStorageFile = resolveIndexFile(libraryRoot);
    if (!Files.isRegularFile(indexStorageFile)) {
      return sourceRepository;
    }
    String json = Files.readString(indexStorageFile, StandardCharsets.UTF_8);
    Object parsedJsonDocument = Json.parseDocument(json);
    if (!(parsedJsonDocument instanceof Map)) {
      throw new IOException(
          "Corrupt index: topLevelJsonObject level is not an object in " + indexStorageFile);
    }
    Map<?, ?> topLevelJsonObject = (Map<?, ?>) parsedJsonDocument;

    Object fileArrayPayload = topLevelJsonObject.get("files");
    if (fileArrayPayload instanceof List) {
      for (Object fileEntryNode : (List<?>) fileArrayPayload) {
        if (!(fileEntryNode instanceof Map)) {
          continue;
        }
        Map<?, ?> m = (Map<?, ?>) fileEntryNode;
        String id = coerceToStringValue(m.get("id"));
        if (id == null || id.isEmpty()) {
          continue;
        }
        FileTagDescriptor f =
            new FileTagDescriptor(
                id,
                extractStringWithFallback(m.get("displayName"), id),
                extractStringWithFallback(m.get("originalRel"), ""),
                extractStringWithFallback(m.get("storedRel"), ""),
                extractLongWithFallback(m.get("size"), 0L),
                extractStringWithFallback(m.get("sha256"), id));
        Object tagObjectPayload = m.get("tags");
        if (tagObjectPayload instanceof Map) {
          Map<String, List<String>> tags = new LinkedHashMap<>();
          for (Map.Entry<?, ?> e : ((Map<?, ?>) tagObjectPayload).entrySet()) {
            String categoryKey = String.valueOf(e.getKey()).trim();
            if (categoryKey.isEmpty()) {
              continue;
            }
            List<String> associatedValueList = new ArrayList<>();
            if (e.getValue() instanceof List) {
              for (Object v : (List<?>) e.getValue()) {
                // Migrate legacy entries: commas always separate associatedValueList.
                for (String part : String.valueOf(v).split(",", -1)) {
                  String val = part.trim();
                  if (!val.isEmpty() && !containsIgnoreCase(associatedValueList, val)) {
                    associatedValueList.add(val);
                  }
                }
              }
            }
            if (!associatedValueList.isEmpty()) {
              tags.put(categoryKey, associatedValueList);
            }
          }
          f.restoreTagSnapshot(tags);
        }
        sourceRepository.registerEntry(f);
      }
    }
    sourceRepository.recordLastSortCategory(
        extractStringWithFallback(topLevelJsonObject.get("lastSortCategory"), ""));
    Object directoryArrayPayload = topLevelJsonObject.get("lastSortDirs");
    if (directoryArrayPayload instanceof List) {
      List<String> previouslySortedDirectories = new ArrayList<>();
      for (Object directoryName : (List<?>) directoryArrayPayload) {
        previouslySortedDirectories.add(String.valueOf(directoryName));
      }
      sourceRepository.recordLastSortDirectories(previouslySortedDirectories);
    }
    return sourceRepository;
  }

  /** Saves the sourceRepository atomically (write temp + move). */
  public static void persistRepository(TagRepository sourceRepository) throws IOException {
    Path libraryRoot = sourceRepository.getLibraryRoot();
    StringBuilder documentBuilder = new StringBuilder(4096);
    documentBuilder.append("{\n");
    documentBuilder.append("  \"version\": ").append(SCHEMA_VERSION).append(",\n");
    documentBuilder
        .append("  \"lastSortCategory\": ")
        .append(Json.escapeForJson(sourceRepository.retrieveLastSortCategory()))
        .append(",\n");
    documentBuilder.append("  \"lastSortDirs\": [");
    List<String> previouslySortedDirectories = sourceRepository.retrieveLastSortDirectories();
    for (int i = 0; i < previouslySortedDirectories.size(); i++) {
      if (i > 0) {
        documentBuilder.append(", ");
      }
      documentBuilder.append(Json.escapeForJson(previouslySortedDirectories.get(i)));
    }
    documentBuilder.append("],\n");
    documentBuilder.append("  \"files\": [\n");
    List<FileTagDescriptor> files = sourceRepository.retrieveAllEntries();
    for (int i = 0; i < files.size(); i++) {
      FileTagDescriptor f = files.get(i);
      documentBuilder.append("    {\n");
      documentBuilder
          .append("      \"id\": ")
          .append(Json.escapeForJson(f.getContentIdentifier()))
          .append(",\n");
      documentBuilder
          .append("      \"displayName\": ")
          .append(Json.escapeForJson(nullToEmptyString(f.getDisplayLabel())))
          .append(",\n");
      documentBuilder
          .append("      \"originalRel\": ")
          .append(Json.escapeForJson(nullToEmptyString(f.getOriginalRelativePath())))
          .append(",\n");
      documentBuilder
          .append("      \"storedRel\": ")
          .append(Json.escapeForJson(nullToEmptyString(f.getStoredRelativePath())))
          .append(",\n");
      documentBuilder.append("      \"size\": ").append(f.getFileSizeInBytes()).append(",\n");
      documentBuilder
          .append("      \"sha256\": ")
          .append(Json.escapeForJson(nullToEmptyString(f.getContentHash())))
          .append(",\n");
      documentBuilder.append("      \"tags\": {");
      List<Map.Entry<String, List<String>>> tagEntries =
          new ArrayList<>(f.retrieveTagSnapshot().entrySet());
      if (!tagEntries.isEmpty()) {
        documentBuilder.append("\n");
        for (int t = 0; t < tagEntries.size(); t++) {
          Map.Entry<String, List<String>> e = tagEntries.get(t);
          documentBuilder.append("        ").append(Json.escapeForJson(e.getKey())).append(": [");
          List<String> associatedValueList = e.getValue();
          for (int v = 0; v < associatedValueList.size(); v++) {
            if (v > 0) {
              documentBuilder.append(", ");
            }
            documentBuilder.append(Json.escapeForJson(associatedValueList.get(v)));
          }
          documentBuilder.append("]");
          documentBuilder.append(t + 1 < tagEntries.size() ? ",\n" : "\n");
        }
        documentBuilder.append("      ");
      }
      documentBuilder.append("}\n");
      documentBuilder.append("    }");
      documentBuilder.append(i + 1 < files.size() ? ",\n" : "\n");
    }
    documentBuilder.append("  ]\n");
    documentBuilder.append("}\n");

    Path destinationIndexFile = resolveIndexFile(libraryRoot);
    Path stagingTempFile =
        destinationIndexFile.resolveSibling(INDEX_STORAGE_FILENAME + ".stagingTempFile");
    Files.writeString(stagingTempFile, documentBuilder.toString(), StandardCharsets.UTF_8);
    try {
      Files.move(
          stagingTempFile,
          destinationIndexFile,
          java.nio.file.StandardCopyOption.ATOMIC_MOVE,
          java.nio.file.StandardCopyOption.REPLACE_EXISTING);
    } catch (java.nio.file.AtomicMoveNotSupportedException e) {
      Files.move(
          stagingTempFile, destinationIndexFile, java.nio.file.StandardCopyOption.REPLACE_EXISTING);
    }
  }

  private static String nullToEmptyString(String sourceText) {
    return sourceText == null ? "" : sourceText;
  }

  private static String coerceToStringValue(Object o) {
    return o == null ? null : String.valueOf(o);
  }

  private static String extractStringWithFallback(Object o, String fallback) {
    return o == null ? fallback : String.valueOf(o);
  }

  private static boolean containsIgnoreCase(List<String> associatedValueList, String candidate) {
    for (String existing : associatedValueList) {
      if (existing.equalsIgnoreCase(candidate)) {
        return true;
      }
    }
    return false;
  }

  private static long extractLongWithFallback(Object o, long fallback) {
    if (o instanceof Number) {
      return ((Number) o).longValue();
    }
    return fallback;
  }

  /** Minimal JSON parser for the subset this app writes. */
  static final class Json {

    private Json() {}

    static String escapeForJson(String sourceText) {
      StringBuilder documentBuilder = new StringBuilder(sourceText.length() + 2);
      documentBuilder.append('"');
      for (int i = 0; i < sourceText.length(); i++) {
        char currentCharacter = sourceText.charAt(i);
        switch (currentCharacter) {
          case '"':
            documentBuilder.append("\\\"");
            break;
          case '\\':
            documentBuilder.append("\\\\");
            break;
          case '\n':
            documentBuilder.append("\\n");
            break;
          case '\r':
            documentBuilder.append("\\r");
            break;
          case '\t':
            documentBuilder.append("\\t");
            break;
          case '\b':
            documentBuilder.append("\\b");
            break;
          case '\f':
            documentBuilder.append("\\f");
            break;
          default:
            if (currentCharacter < 0x20) {
              documentBuilder.append(String.format("\\u%04x", (int) currentCharacter));
            } else {
              documentBuilder.append(currentCharacter);
            }
        }
      }
      documentBuilder.append('"');
      return documentBuilder.toString();
    }

    static Object parseDocument(String text) throws IOException {
      Parser p = new Parser(text);
      Object value = p.readNextValue();
      p.skipInsignificantWhitespace();
      if (!p.isInputExhausted()) {
        throw new IOException("Corrupt index: trailing characters");
      }
      return value;
    }

    private static final class Parser {
      private final String sourceText;
      private int currentParsePosition;

      Parser(String sourceText) {
        this.sourceText = sourceText;
      }

      boolean isInputExhausted() {
        return currentParsePosition >= sourceText.length();
      }

      void skipInsignificantWhitespace() {
        while (!isInputExhausted()) {
          char currentCharacter = sourceText.charAt(currentParsePosition);
          if (currentCharacter == ' '
              || currentCharacter == '\t'
              || currentCharacter == '\n'
              || currentCharacter == '\r') {
            currentParsePosition++;
          } else {
            break;
          }
        }
      }

      Object readNextValue() throws IOException {
        skipInsignificantWhitespace();
        if (isInputExhausted()) {
          throw new IOException("Corrupt index: unexpected end of file");
        }
        char currentCharacter = sourceText.charAt(currentParsePosition);
        switch (currentCharacter) {
          case '{':
            return readJsonObject();
          case '[':
            return readJsonArray();
          case '"':
            return readJsonString();
          case 't':
            consumeExpectedKeyword("true");
            return Boolean.TRUE;
          case 'f':
            consumeExpectedKeyword("false");
            return Boolean.FALSE;
          case 'n':
            consumeExpectedKeyword("null");
            return null;
          default:
            return readJsonNumber();
        }
      }

      Map<String, Object> readJsonObject() throws IOException {
        Map<String, Object> objectAccumulator = new LinkedHashMap<>();
        currentParsePosition++; // {
        skipInsignificantWhitespace();
        if (!isInputExhausted() && sourceText.charAt(currentParsePosition) == '}') {
          currentParsePosition++;
          return objectAccumulator;
        }
        while (true) {
          skipInsignificantWhitespace();
          if (isInputExhausted() || sourceText.charAt(currentParsePosition) != '"') {
            throw new IOException("Corrupt index: expected string memberName");
          }
          String memberName = readJsonString();
          skipInsignificantWhitespace();
          if (isInputExhausted() || sourceText.charAt(currentParsePosition) != ':') {
            throw new IOException("Corrupt index: expected ':'");
          }
          currentParsePosition++;
          objectAccumulator.put(memberName, readNextValue());
          skipInsignificantWhitespace();
          if (isInputExhausted()) {
            throw new IOException("Corrupt index: unterminated object");
          }
          char delimiterCharacter = sourceText.charAt(currentParsePosition++);
          if (delimiterCharacter == '}') {
            return objectAccumulator;
          }
          if (delimiterCharacter != ',') {
            throw new IOException("Corrupt index: expected ',' or '}'");
          }
        }
      }

      List<Object> readJsonArray() throws IOException {
        List<Object> arrayAccumulator = new ArrayList<>();
        currentParsePosition++; // [
        skipInsignificantWhitespace();
        if (!isInputExhausted() && sourceText.charAt(currentParsePosition) == ']') {
          currentParsePosition++;
          return arrayAccumulator;
        }
        while (true) {
          arrayAccumulator.add(readNextValue());
          skipInsignificantWhitespace();
          if (isInputExhausted()) {
            throw new IOException("Corrupt index: unterminated array");
          }
          char delimiterCharacter = sourceText.charAt(currentParsePosition++);
          if (delimiterCharacter == ']') {
            return arrayAccumulator;
          }
          if (delimiterCharacter != ',') {
            throw new IOException("Corrupt index: expected ',' or ']'");
          }
        }
      }

      String readJsonString() throws IOException {
        currentParsePosition++; // opening quote
        StringBuilder documentBuilder = new StringBuilder();
        while (true) {
          if (isInputExhausted()) {
            throw new IOException("Corrupt index: unterminated string");
          }
          char currentCharacter = sourceText.charAt(currentParsePosition++);
          if (currentCharacter == '"') {
            return documentBuilder.toString();
          }
          if (currentCharacter == '\\') {
            if (isInputExhausted()) {
              throw new IOException("Corrupt index: bad escape");
            }
            char e = sourceText.charAt(currentParsePosition++);
            switch (e) {
              case '"':
                documentBuilder.append('"');
                break;
              case '\\':
                documentBuilder.append('\\');
                break;
              case '/':
                documentBuilder.append('/');
                break;
              case 'n':
                documentBuilder.append('\n');
                break;
              case 'r':
                documentBuilder.append('\r');
                break;
              case 't':
                documentBuilder.append('\t');
                break;
              case 'b':
                documentBuilder.append('\b');
                break;
              case 'f':
                documentBuilder.append('\f');
                break;
              case 'u':
                if (currentParsePosition + 4 > sourceText.length()) {
                  throw new IOException("Corrupt index: bad unicode escape");
                }
                try {
                  int codePointValue =
                      Integer.parseInt(
                          sourceText.substring(currentParsePosition, currentParsePosition + 4), 16);
                  documentBuilder.append((char) codePointValue);
                  currentParsePosition += 4;
                } catch (NumberFormatException parsingFailure) {
                  throw new IOException("Corrupt index: bad unicode escape");
                }
                break;
              default:
                throw new IOException("Corrupt index: bad escape \\" + e);
            }
          } else {
            documentBuilder.append(currentCharacter);
          }
        }
      }

      Number readJsonNumber() throws IOException {
        int start = currentParsePosition;
        while (!isInputExhausted()) {
          char currentCharacter = sourceText.charAt(currentParsePosition);
          if ((currentCharacter >= '0' && currentCharacter <= '9')
              || currentCharacter == '-'
              || currentCharacter == '+'
              || currentCharacter == '.'
              || currentCharacter == 'e'
              || currentCharacter == 'E') {
            currentParsePosition++;
          } else {
            break;
          }
        }
        String numericToken = sourceText.substring(start, currentParsePosition);
        try {
          return Long.parseLong(numericToken);
        } catch (NumberFormatException e) {
          try {
            return Double.parseDouble(numericToken);
          } catch (NumberFormatException e2) {
            throw new IOException("Corrupt index: bad number " + numericToken);
          }
        }
      }

      void consumeExpectedKeyword(String expectedKeyword) throws IOException {
        if (sourceText.startsWith(expectedKeyword, currentParsePosition)) {
          currentParsePosition += expectedKeyword.length();
        } else {
          throw new IOException("Corrupt index: expected " + expectedKeyword);
        }
      }
    }
  }
}
