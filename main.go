package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
)

type SxGeo struct {
	fileBytes              []byte
	headerBytes            []byte
	firstBlockBytes        []byte
	mainIndexElementsBytes []byte
	dbBytes                []byte

	version                int
	charset                int
	firstElementIndexCount int
	mainIndexElementsCount uint32
	oneIndexBlocksCount    uint32
	rangesCount            uint32
	idBlockSize            uint32
	maxRegionRecordSize    uint16
	maxCityRecordSize      uint16
	regionsLibrarySize     uint32
	citiesLibrarySize      uint32
	countriesLibrarySize   uint32
	maxCountryRecordSize   uint16
	packSize               uint16

	blockSize  uint32
	countryMap []string
}

func (sxGeo *SxGeo) GetCountryByIp(ip string) string {
	ipInt, _ := strconv.ParseUint(strings.Split(ip, ".")[0], 10, 32)
	firstIpByteInt := ip2Long(ip)

	firstIpByte := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(firstIpByte, firstIpByteInt)

	searchStarts := (ipInt - 1) * 4
	blocks := sxGeo.firstBlockBytes[searchStarts : searchStarts+8]
	blockInts := []uint32{
		binary.BigEndian.Uint32(blocks[0:4]),
		binary.BigEndian.Uint32(blocks[4:8]),
	}

	var min uint32
	var max uint32
	var part uint32

	if blockInts[0]-blockInts[1] > sxGeo.oneIndexBlocksCount {
		part = sxGeo.searchBlockInMainIndex(
			firstIpByte,
			blockInts[0]/sxGeo.oneIndexBlocksCount,
			blockInts[1]/sxGeo.oneIndexBlocksCount-1,
		)

		if part > 0 {
			min = part * sxGeo.oneIndexBlocksCount
		} else {
			min = 0
		}
		if part > sxGeo.mainIndexElementsCount {
			max = sxGeo.rangesCount
		} else {
			max = (part + 1) * sxGeo.oneIndexBlocksCount
		}
		if min < blockInts[0] {
			min = blockInts[0]
		}
		if max > blockInts[1] {
			max = blockInts[1]
		}
	} else {
		min = blockInts[0]
		max = blockInts[1]
	}

	if max-min > 1 {
		nFIP := firstIpByte[1:]
		for true {
			if max-min <= 8 {
				break
			}
			offset := (min + max) >> 1
			if bytes.Compare(nFIP, sxGeo.dbBytes[offset*sxGeo.blockSize:offset*sxGeo.blockSize+3]) > 0 {
				min = offset
			} else {
				max = offset
			}
		}
		for true {
			if bytes.Compare(nFIP, sxGeo.dbBytes[min*sxGeo.blockSize:min*sxGeo.blockSize+3]) < 0 {
				break
			}
			min = min + 1
			if min >= max {
				break
			}
		}

	}

	countryFrom := min*sxGeo.blockSize - sxGeo.idBlockSize
	countryTo := countryFrom + sxGeo.idBlockSize

	countryId := int(sxGeo.dbBytes[countryFrom:countryTo][0])
	return sxGeo.getIsoById(countryId)
}

func (sxGeo SxGeo) getIsoById(countryId int) string {
	return sxGeo.countryMap[countryId]
}

func (sxGeo SxGeo) searchBlockInMainIndex(firstIpByte []byte, min uint32, max uint32) uint32 {
	for true {
		if max-min <= 8 {
			break
		}
		offset := min - max>>1

		if bytes.Compare(firstIpByte, sxGeo.mainIndexElementsBytes[offset*4:offset*4+4]) > 0 {
			min = offset
		} else {
			max = offset
		}
	}
	for true {
		fromSlice := min * 4
		toSlice := fromSlice + 4
		newSlice := sxGeo.mainIndexElementsBytes[fromSlice:toSlice]
		if binary.BigEndian.Uint32(firstIpByte) <= binary.BigEndian.Uint32(newSlice) {
			break
		}
		min = min + 1
		if min >= max {
			break
		}
	}
	return min
}

func NewSxGeo(fileBytes []byte) *SxGeo {
	firstElementIndexCount := int(fileBytes[10:11][0])
	firstBytesEnds := 40 + (firstElementIndexCount * 4)

	mainIndexElementsCount := uint32(binary.BigEndian.Uint16(fileBytes[11:13]))
	mainIndexElementsEnds := uint32(firstBytesEnds) + (mainIndexElementsCount * 4)

	rangesCount := binary.BigEndian.Uint32(fileBytes[15:19])

	idBlockSize := uint32(int(fileBytes[19:20][0]))
	blockSize := 3 + idBlockSize

	dbEnds := rangesCount * uint32(blockSize)

	return &SxGeo{
		fileBytes:              fileBytes,
		version:                int(fileBytes[3:4][0]),
		charset:                int(fileBytes[9:10][0]),
		firstElementIndexCount: firstElementIndexCount,
		mainIndexElementsCount: mainIndexElementsCount,
		oneIndexBlocksCount:    uint32(binary.BigEndian.Uint16(fileBytes[13:15])),
		rangesCount:            rangesCount,
		idBlockSize:            idBlockSize,
		maxRegionRecordSize:    binary.BigEndian.Uint16(fileBytes[20:22]),
		maxCityRecordSize:      binary.BigEndian.Uint16(fileBytes[22:24]),
		regionsLibrarySize:     binary.BigEndian.Uint32(fileBytes[24:28]),
		citiesLibrarySize:      binary.BigEndian.Uint32(fileBytes[28:32]),
		maxCountryRecordSize:   binary.BigEndian.Uint16(fileBytes[32:34]),
		countriesLibrarySize:   binary.BigEndian.Uint32(fileBytes[34:38]),
		packSize:               binary.BigEndian.Uint16(fileBytes[38:40]),

		blockSize: blockSize,

		headerBytes:            fileBytes[:40],
		firstBlockBytes:        fileBytes[40:firstBytesEnds],
		mainIndexElementsBytes: fileBytes[firstBytesEnds:mainIndexElementsEnds],
		dbBytes:                fileBytes[mainIndexElementsEnds:dbEnds],

		countryMap: []string{
			"", "AP", "EU", "AD", "AE", "AF", "AG", "AI", "AL", "AM", "CW", "AO", "AQ", "AR", "AS", "AT", "AU",
			"AW", "AZ", "BA", "BB", "BD", "BE", "BF", "BG", "BH", "BI", "BJ", "BM", "BN", "BO", "BR", "BS",
			"BT", "BV", "BW", "BY", "BZ", "CA", "CC", "CD", "CF", "CG", "CH", "CI", "CK", "CL", "CM", "CN",
			"CO", "CR", "CU", "CV", "CX", "CY", "CZ", "DE", "DJ", "DK", "DM", "DO", "DZ", "EC", "EE", "EG",
			"EH", "ER", "ES", "ET", "FI", "FJ", "FK", "FM", "FO", "FR", "SX", "GA", "GB", "GD", "GE", "GF",
			"GH", "GI", "GL", "GM", "GN", "GP", "GQ", "GR", "GS", "GT", "GU", "GW", "GY", "HK", "HM", "HN",
			"HR", "HT", "HU", "ID", "IE", "IL", "IN", "IO", "IQ", "IR", "IS", "IT", "JM", "JO", "JP", "KE",
			"KG", "KH", "KI", "KM", "KN", "KP", "KR", "KW", "KY", "KZ", "LA", "LB", "LC", "LI", "LK", "LR",
			"LS", "LT", "LU", "LV", "LY", "MA", "MC", "MD", "MG", "MH", "MK", "ML", "MM", "MN", "MO", "MP",
			"MQ", "MR", "MS", "MT", "MU", "MV", "MW", "MX", "MY", "MZ", "NA", "NC", "NE", "NF", "NG", "NI",
			"NL", "NO", "NP", "NR", "NU", "NZ", "OM", "PA", "PE", "PF", "PG", "PH", "PK", "PL", "PM", "PN",
			"PR", "PS", "PT", "PW", "PY", "QA", "RE", "RO", "RU", "RW", "SA", "SB", "SC", "SD", "SE", "SG",
			"SH", "SI", "SJ", "SK", "SL", "SM", "SN", "SO", "SR", "ST", "SV", "SY", "SZ", "TC", "TD", "TF",
			"TG", "TH", "TJ", "TK", "TM", "TN", "TO", "TL", "TR", "TT", "TV", "TW", "TZ", "UA", "UG", "UM",
			"US", "UY", "UZ", "VA", "VC", "VE", "VG", "VI", "VN", "VU", "WF", "WS", "YE", "YT", "RS", "ZA",
			"ZM", "ME", "ZW", "A1", "XK", "O1", "AX", "GG", "IM", "JE", "BL", "MF", "BQ", "SS",
		},
	}
}

func ip2Long(ip string) uint32 {
	var long uint32
	binary.Read(bytes.NewBuffer(net.ParseIP(ip).To4()), binary.BigEndian, &long)
	return long
}
