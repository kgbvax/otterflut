

const __constant unsigned int hexval32[256]=
        {['0']= 0, ['1']= 1, ['2']= 2, ['3']= 3, ['4']= 4, ['5']= 5,
        ['6']= 6, ['7']= 7, ['8']= 8, ['9']= 9, ['a']= 0xA, ['A']= 0xA,
        ['b']= 0xB, ['B']= 0xB, ['c']=0xC, ['C']=0xC, ['d']= 0xD, ['D']= 0xD,
        ['e']= 0xE, ['E']= 0xE, ['f']= 0xF, ['F']= 0xF};

const __constant int elemSize=32;

int searchNonSpace(__global uchar *line,int offset);
int searchSpace(__global uchar *line,int offset);
ushort parsUint(__global uchar *m, int len);
uint parseHex3ToBGR(__global uchar *m);


inline int searchNonSpace(__global uchar *line,int offset) {
    int i=offset;
    while (i < elemSize &&  line[i] == ' ' && line[i]!=0 ) {
        i++;
    }
    return i;
}

inline int searchSpace(__global uchar *line,int offset) {
    int i=offset;
    while (i < elemSize &&  line[i] != ' ' && line[i]!=0 ) {
           i++;
    }
    return i;
}


inline ushort  parsUint(__global uchar *m, int len) {
	const ushort ZERO = 0x30;

	switch (len) {
	case 3: //assumed to be the  most likely case
		return  100*(m[0]-ZERO) +
				10*(m[1]-ZERO) +
				m[2]-ZERO;
	case 4:
		return  1000*(m[0]-ZERO) +
				100*(m[1]-ZERO) +
				10*(m[2]-ZERO) +
				m[3]-ZERO;
	case 2:
		return	10*(m[0]-ZERO) +
				m[1]-ZERO;
	case 1:  //least likely case
		return  m[0] - ZERO;
	}

	//todo increment error count
	return 0;
}

inline uint parseHex3ToBGR(__global uchar *m)  {
	//MUL version, compiles to shifts
	// BB GG RR
	//printf("%u %u %u %u\n",(uint)m[0],(uint)m[1],(uint)m[2],(uint)m[3],(uint)m[4],(uint)m[5]);

	return 0x100000*hexval32[m[4]] + 0x010000*hexval32[m[5]] + 0x001000*hexval32[m[2]] +
           0x000100*hexval32[m[3]] + 0x000010*hexval32[m[0]] + hexval32[m[1]];
}

kernel void clparser(global uchar *lines,  global ushort *x,  global ushort *y,  global uint *color) {
  // int offset=0;
 //   int len=0;
    int idx = get_global_id(0);

    __global uchar *line =&(lines[idx*elemSize]);


    int startOfX = searchNonSpace(line,3);
   // printf("startOfX=%d\n",startOfX);

    int endOfX = searchSpace(line,startOfX+1);
  //  printf("endOfX=%d\n",endOfX);


  //  printf("len=%d\n",len);
   // printf("start ch0=%hhx",line[startOfX]);
   // printf("start ch1=%hhx",line[startOfX+1]);
   //  printf("start ch2=%hhx",line[startOfX+2]);


    x[idx]=parsUint(&line[startOfX],endOfX-startOfX);;
  //  printf("parsed %d",parsed);
     int startOfY =endOfX+1;
    int endOfY = searchSpace(line,startOfY);

    y[idx]=parsUint(&line[startOfY],endOfY-startOfY);;


    int startOfC = endOfY+1;
   // int endOfC = searchSpace(line,startOfC);
   // printf("startOfC %d\n",startOfC);
   // printf("endOfC %d\n",endOfC);
     //printf(( __constant char *)"parsi %d\n",idx);


   // printf("col=%d %d\n",idx,c);
    color[idx]=parseHex3ToBGR(&(line[startOfC]));


    // PX 123 34 FFFFFF

  //  searchNonSpace
  //  parseUint

}
