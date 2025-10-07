// NavControlSystem/pkg/logger/logger.go
package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rackov/NavControl/pkg/config"
	"github.com/rackov/NavControl/proto"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogEntry представляет одну запись в логе
type LogEntry struct {
	File    string    `json:"file"`
	Level   string    `json:"level"`
	Message string    `json:"msg"`
	Time    time.Time `json:"time"`
	// Другие поля, которые могут быть в логе
}

// UnmarshalJSON кастомный метод для парсинга JSON с нестандартным форматом времени
func (e *LogEntry) UnmarshalJSON(data []byte) error {
	// Создаем псевдоним типа, чтобы избежать рекурсии при вызове json.Unmarshal
	type Alias LogEntry

	// Временная структура для парсинга с полем time в виде строки
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	// Парсим JSON во временную структуру
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Парсим время из строки в формате "2006-01-02 15:04:05"
	if aux.Time != "" {
		parsedTime, err := time.Parse("2006-01-02 15:04:05", aux.Time)
		if err != nil {
			return fmt.Errorf("failed to parse time: %v", err)
		}
		e.Time = parsedTime
	}

	return nil
}

// Logger - это наша основная структура, которая инкапсулирует logrus и lumberjack.
// Мы экспортируем ее, чтобы пользователь мог работать с экземпляром.
type Logger struct {
	*logrus.Logger                    // Встраиваем стандартный логгер logrus
	fileHook       *lumberjack.Logger // Сохраняем ссылку на lumberjack для доступа к Sync()
	logPath        string
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// NewLogger создает и возвращает новый настроенный экземпляр Logger.
// Это основная функция для создания логгера.
func NewLogger(cfg config.ConfigLog) (*Logger, error) {
	var l *Logger
	var err error

	once.Do(func() {
		// 1. Создаем lumberjack для ротации логов
		fileHook := &lumberjack.Logger{
			Filename:   cfg.LogFilePath,
			MaxSize:    cfg.MaxSize,    // megabytes
			MaxBackups: cfg.MaxBackups, // files
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // compress
		}

		// 2. Создаем стандартный логгер logrus
		baseLogger := logrus.New()

		// 3. Устанавливаем уровень логирования
		level, err := logrus.ParseLevel(cfg.LogLevel)
		if err != nil {
			level = logrus.InfoLevel
			fmt.Fprintf(os.Stderr, "Failed to parse log level '%s', defaulting to 'info'. Error: %v\n", cfg.LogLevel, err)
		}
		baseLogger.SetLevel(level)

		// 4. Устанавливаем форматтер
		baseLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
				return "", fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
			},
		})
		baseLogger.SetReportCaller(true)

		// 5. Устанавливаем вывод. lumberjack будет основным.
		// Можно добавить вывод в консоль для разработки, раскомментировав строку ниже.
		baseLogger.SetOutput(fileHook)
		baseLogger.SetOutput(io.MultiWriter(fileHook, os.Stdout))

		// 6. Создаем наш экземпляр Logger
		l = &Logger{
			Logger:   baseLogger,
			fileHook: fileHook,
			logPath:  cfg.LogFilePath,
		}

		defaultLogger = l // Сохраняем как "логгер по умолчанию"
	})

	if l == nil {
		return nil, fmt.Errorf("failed to initialize logger after Do block")
	}
	return l, err
}

// Sync сбрасывает буферы на диск. Реализует интерфейс syncer.
// Теперь это метод нашего Logger, что делает API последовательным.
func (l *Logger) Close() error {
	if l.fileHook != nil {
		return l.fileHook.Close()
	}
	return nil
}

// GetLevel возвращает текущий уровень логирования
func (l *Logger) GetLevel() string {
	return l.Logger.GetLevel().String()
}

// SetLevel динамически изменяет уровень логирования.
// Также сделано методом для работы с экземпляром.
func (l *Logger) SetLevel(levelStr string) error {
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", levelStr, err)
	}
	l.Logger.SetLevel(level)
	l.Infof("Log level changed to %s", level.String())
	return nil
}

// Default возвращает "логгер по умолчанию".
// Полезно, если не хочется передавать логгер через всю иерархию вызовов.
func Default() *Logger {
	if defaultLogger == nil {
		// Можно вернуть логгер с настройками по умолчанию, если он еще не был создан
		// или паниковать, это зависит от политики проекта.
		panic("Logger not initialized. Call NewLogger first.")
	}
	return defaultLogger
}

type Filter struct {
	Level     string
	StartDate int64
	EndDate   int64
	Limit     int32
	IdSrv     int32
	Port      int32
	Protocol  string
	PosEnd    bool
}

// ReadLogs читает логи с применением фильтров
func (l *Logger) ReadLogs(level string, startDate, endDate int64, limit int32) ([]string, error) {
	// Открываем файл логов
	file, err := os.Open(l.logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string
	var count int32

	// Преобразуем timestamp в time.Time
	var startTime, endTime time.Time
	if startDate != 0 {
		startTime = time.Unix(startDate, 0)
	}
	if endDate != 0 {
		endTime = time.Unix(endDate, 0)
	}

	// Создаем сканер для чтения файла
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Парсим JSON строку
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Если не удалось распарсить как JSON, пропускаем строку
			continue
		}

		// Фильтр по уровню
		if level != "" && !strings.EqualFold(entry.Level, level) {
			continue
		}

		// Фильтр по начальной дате
		if !startTime.IsZero() && entry.Time.Before(startTime) {
			continue
		}

		// Фильтр по конечной дате
		if !endTime.IsZero() && entry.Time.After(endTime) {
			continue
		}

		// Добавляем строку в результат
		result = append(result, line)
		count++

		// Проверяем лимит
		if limit > 0 && count >= limit {
			break
		}
	}

	// Проверяем ошибки сканирования
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// pkg/logger/logger.go

// ReadLogs читает логи с применением фильтров
func (l *Logger) ReadLogEnds(level string, startDate, endDate int64, limit int32) ([]string, error) {
	// Открываем файл логов
	file, err := os.Open(l.logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Получаем информацию о файле
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Получаем размер файла
	fileSize := fileInfo.Size()

	// Если файл пустой, возвращаем пустой результат
	if fileSize == 0 {
		return []string{}, nil
	}

	// Устанавливаем позицию для чтения с конца файла
	// Начнем с последних 2 КБ для поиска начала последней строки
	position := fileSize - 1
	if position < 0 {
		position = 0
	}

	// Буфер для чтения файла
	buf := make([]byte, 1)

	// Ищем начало последней строки
	for position > 0 {
		// Читаем байт с текущей позиции
		_, err := file.ReadAt(buf, position)
		if err != nil {
			return nil, err
		}

		// Если нашли символ новой строки, это начало последней строки
		if buf[0] == '\n' {
			break
		}

		// Двигаемся к предыдущему байту
		position--
	}

	// Теперь позиция указывает на начало последней строки или на начало файла

	var result []string
	var count int32

	// Преобразуем timestamp в time.Time
	var startTime, endTime time.Time
	if startDate != 0 {
		startTime = time.Unix(startDate, 0)
	}
	if endDate != 0 {
		endTime = time.Unix(endDate, 0)
	}

	// Создаем сканер для чтения файла с указанной позиции
	section := io.NewSectionReader(file, position, fileSize-position)
	scanner := bufio.NewScanner(section)

	// Если мы не в начале файла, пропускаем первую строку (она может быть неполной)
	if position > 0 {
		scanner.Scan()
	}

	// Читаем строки в обратном порядке
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Обрабатываем строки в обратном порядке
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		// Парсим JSON строку с использованием кастомного парсера
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Если не удалось распарсить как JSON, пропускаем строку
			return nil, err
			// continue
		}

		// Фильтр по уровню
		if level != "" && !strings.EqualFold(entry.Level, level) {
			continue
		}

		// Фильтр по начальной дате
		if !startTime.IsZero() && entry.Time.Before(startTime) {
			continue
		}

		// Фильтр по конечной дате
		if !endTime.IsZero() && entry.Time.After(endTime) {
			continue
		}

		// Добавляем строку в результат
		result = append(result, line)
		count++

		// Проверяем лимит
		if limit > 0 && count >= limit {
			break
		}
	}

	// Проверяем ошибки сканирования
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func StrToLoglevel(s string) (logLevel proto.LogLevel) {
	switch s {
	case "debug":
		logLevel = proto.LogLevel_DEBUG
	case "info":
		logLevel = proto.LogLevel_INFO
	case "warn":
		logLevel = proto.LogLevel_WARN
	case "error":
		logLevel = proto.LogLevel_ERROR
	default:
		logLevel = proto.LogLevel_INFO // значение по умолчанию
	}
	return
}
