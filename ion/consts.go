package ion

import (
	"reflect"
	"time"
)

var binaryNulls = func() []byte {
	ret := make([]byte, StructType+1)
	ret[NoType] = 0x0F
	ret[NullType] = 0x0F
	ret[BoolType] = 0x1F
	ret[IntType] = 0x2F
	ret[FloatType] = 0x4F
	ret[DecimalType] = 0x5F
	ret[TimestampType] = 0x6F
	ret[SymbolType] = 0x7F
	ret[StringType] = 0x8F
	ret[ClobType] = 0x9F
	ret[BlobType] = 0xAF
	ret[ListType] = 0xBF
	ret[SexpType] = 0xCF
	ret[StructType] = 0xDF
	return ret
}()

var textNulls []string = func() []string {
	ret := make([]string, StructType+1)
	ret[NoType] = "null"
	ret[NullType] = "null.null"
	ret[BoolType] = "null.bool"
	ret[IntType] = "null.int"
	ret[FloatType] = "null.float"
	ret[DecimalType] = "null.decimal"
	ret[TimestampType] = "null.timestamp"
	ret[SymbolType] = "null.symbol"
	ret[StringType] = "null.string"
	ret[ClobType] = "null.clob"
	ret[BlobType] = "null.blob"
	ret[ListType] = "null.list"
	ret[SexpType] = "null.sexp"
	ret[StructType] = "null.struct"
	return ret
}()

var hexChars = []byte{
	'0', '1', '2', '3', '4', '5', '6', '7',
	'8', '9', 'A', 'B', 'C', 'D', 'E', 'F',
}

var timeType = reflect.TypeOf(time.Time{})
var decimalType = reflect.TypeOf(Decimal{})
